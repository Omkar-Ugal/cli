// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package tunnel

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newLocalTLSListener starts a TLS listener on 127.0.0.1 backed by an
// ephemeral, self-signed certificate, standing in for the remote tunnel
// service that tunnelConnection.handle dials via dialRemote.
func newLocalTLSListener(t *testing.T) net.Listener {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	l, err := tls.Listen("tcp4", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })
	return l
}

// setupHandleTest starts a fake remote tunnel-service peer (running
// remoteHandler against the accepted connection) and returns a
// tunnelConnection wired up to dial it, plus the local ("client") end of the
// net.Pipe that stands in for the accepted local connection normally handed
// to handle by tunnelRelay.up.
func setupHandleTest(t *testing.T, remoteHandler func(t *testing.T, rc net.Conn)) (*tunnelConnection, net.Conn) {
	t.Helper()
	l := newLocalTLSListener(t)

	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		rc, err := l.Accept()
		if err != nil {
			return
		}
		defer rc.Close()
		remoteHandler(t, rc)
	}()
	t.Cleanup(func() { <-acceptDone })

	clientSide, connSide := net.Pipe()
	t.Cleanup(func() { clientSide.Close() })

	c := &tunnelConnection{
		relay: &tunnelRelay{remoteAddr: l.Addr().String(), insecure: true},
		conn:  connSide,
	}
	return c, clientSide
}

// readAuth and writeStatus are invoked from goroutines that stand in for the
// remote peer (directly, and indirectly via setupHandleTest's remoteHandler),
// so they report failures via t.Errorf rather than testify's assert/require:
// require's t.FailNow() is only safe to call from the goroutine running the
// test function, and testifylint otherwise insists error checks use require.
func readAuth(t *testing.T, rc net.Conn, auth []byte) {
	t.Helper()
	got := make([]byte, len(auth))
	if _, err := io.ReadFull(rc, got); err != nil {
		t.Errorf("reading auth: %v", err)
		return
	}
	if !bytes.Equal(auth, got) {
		t.Errorf("auth = %q, want %q", got, auth)
	}
}

func writeStatus(t *testing.T, rc net.Conn, status int16) {
	t.Helper()
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, status); err != nil {
		t.Errorf("encoding status: %v", err)
		return
	}
	if _, err := rc.Write(buf.Bytes()); err != nil {
		t.Errorf("writing status: %v", err)
	}
}

// TestTunnelConnectionHandle exercises tunnelConnection.handle's auth-status
// branches and the bidirectional data relay. Run with -race: the "relays
// data both ways" subtest races the connection's two copy directions against
// each other on purpose, which is exactly the scenario that a regression of
// the shared-err data race (a torn error value written concurrently by both
// io.Copy directions) would be caught by.
func TestTunnelConnectionHandle(t *testing.T) {
	auth := []byte("test-auth-cookie")

	t.Run("status zero: no available connections", func(t *testing.T) {
		c, _ := setupHandleTest(t, func(t *testing.T, rc net.Conn) {
			readAuth(t, rc, auth)
			writeStatus(t, rc, 0)
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			c.handle(context.Background(), auth, "my-instance", "my-instance:8080")
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("handle did not return for a zero auth status")
		}
	})

	t.Run("status negative: internal tunnel error", func(t *testing.T) {
		c, _ := setupHandleTest(t, func(t *testing.T, rc net.Conn) {
			readAuth(t, rc, auth)
			writeStatus(t, rc, -1)
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			c.handle(context.Background(), auth, "my-instance", "my-instance:8080")
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("handle did not return for a negative auth status")
		}
	})

	t.Run("remote closes before sending a full status", func(t *testing.T) {
		c, _ := setupHandleTest(t, func(t *testing.T, rc net.Conn) {
			readAuth(t, rc, auth)
			_, _ = rc.Write([]byte{0x01}) // one byte, then hang up early
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			c.handle(context.Background(), auth, "my-instance", "my-instance:8080")
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("handle did not return after a short status read")
		}
	})

	t.Run("status positive: relays data both ways and unblocks on remote close", func(t *testing.T) {
		fromClient := []byte("hello-from-client")
		fromRemote := []byte("hello-from-remote")

		var gotFromClient []byte
		c, clientSide := setupHandleTest(t, func(t *testing.T, rc net.Conn) {
			readAuth(t, rc, auth)
			writeStatus(t, rc, 1)

			_, err := rc.Write(fromRemote)
			require.NoError(t, err)

			buf := make([]byte, len(fromClient))
			_, err = io.ReadFull(rc, buf)
			require.NoError(t, err)
			gotFromClient = buf
			// Returning here closes rc (deferred by the caller), which races
			// handle's two io.Copy directions against each other.
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			c.handle(context.Background(), auth, "my-instance", "my-instance:8080")
		}()

		gotFromRemote := make([]byte, len(fromRemote))
		_, err := io.ReadFull(clientSide, gotFromRemote)
		require.NoError(t, err)
		assert.Equal(t, fromRemote, gotFromRemote)

		_, err = clientSide.Write(fromClient)
		require.NoError(t, err)

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("handle did not return after the remote closed the connection")
		}
		assert.Equal(t, fromClient, gotFromClient)
	})

	t.Run("both ends close concurrently without racing (regression for shared err)", func(t *testing.T) {
		// Reproduces a connection torn down from both ends at roughly the
		// same time (e.g. a mid-transfer reset). Neither close is caused by
		// the other, so if handle's two io.Copy directions still wrote to a
		// shared `err` variable instead of independent ones, this is exactly
		// the scenario -race would flag.
		l := newLocalTLSListener(t)

		rcCh := make(chan net.Conn, 1)
		acceptDone := make(chan struct{})
		go func() {
			defer close(acceptDone)
			rc, err := l.Accept()
			if err != nil {
				return
			}
			readAuth(t, rc, auth)
			writeStatus(t, rc, 1)
			rcCh <- rc
		}()
		t.Cleanup(func() { <-acceptDone })

		clientSide, connSide := net.Pipe()
		c := &tunnelConnection{
			relay: &tunnelRelay{remoteAddr: l.Addr().String(), insecure: true},
			conn:  connSide,
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			c.handle(context.Background(), auth, "my-instance", "my-instance:8080")
		}()

		rc := <-rcCh

		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); rc.Close() }()
		go func() { defer wg.Done(); clientSide.Close() }()
		wg.Wait()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("handle did not return after both ends closed concurrently")
		}
	})

	t.Run("no auth configured: skips the handshake and relays immediately", func(t *testing.T) {
		fromRemote := []byte("hello-from-remote")
		c, clientSide := setupHandleTest(t, func(t *testing.T, rc net.Conn) {
			_, err := rc.Write(fromRemote)
			require.NoError(t, err)
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			c.handle(context.Background(), nil, "my-instance", "my-instance:8080")
		}()

		got := make([]byte, len(fromRemote))
		_, err := io.ReadFull(clientSide, got)
		require.NoError(t, err)
		assert.Equal(t, fromRemote, got)

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("handle did not return after the remote closed the connection")
		}
	})
}
