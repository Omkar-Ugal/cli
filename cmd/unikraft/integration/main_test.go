// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mitchellh/copystructure"
	"github.com/stretchr/testify/require"

	"unikraft.com/x/log"

	"unikraft.com/cli/internal/cmd"
	"unikraft.com/cli/internal/config"
	integ "unikraft.com/cli/internal/integration"
	"unikraft.com/cli/internal/resource"
)

var (
	uniqSeed    string
	uniqCounter atomic.Uint64
)

func init() {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	uniqSeed = hex.EncodeToString(b[:])
}

// uniq generates a unique 12-character hex string for use in test resource names.
// It combines a per-run random prefix with an atomic counter to guarantee
// no duplicates within a single test run.
func uniq() string {
	n := uniqCounter.Add(1)
	return fmt.Sprintf("%s%06x", uniqSeed, n)
}

// testRunner holds state for running integration tests.
type testRunner struct {
	cfg *integ.Config
	*integ.TestEnv
}

// newIntegrationRunner creates a shared runner for integration tests.
// It builds the unikraft binary and loads the integration config.
func newIntegrationRunner(t *testing.T) *testRunner {
	t.Helper()
	integ.SkipUnlessIntegration(t)
	unikraftPath := integ.BuildUnikraftBinary(t)

	cfg, err := integ.LoadConfig(t)
	require.NoError(t, err)

	return &testRunner{
		cfg: cfg,
		TestEnv: &integ.TestEnv{
			UnikraftPath: unikraftPath,
		},
	}
}

// runner creates a new per-subtest runner. If online is true, the test is
// skipped when no integration config is available.
func (ir *testRunner) runner(t *testing.T, online bool) *testRunner {
	t.Helper()
	t.Parallel()

	if online && ir.cfg == nil {
		t.Skip("online test requires config, but no config found")
	}

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	var testCfg *integ.Config
	if ir.cfg != nil {
		cloned, err := copystructure.Copy(ir.cfg)
		require.NoError(t, err)
		testCfg = cloned.(*integ.Config)
		testCfg.Config.Path = configPath
		require.NoError(t, testCfg.Config.Save())
	}

	ctx := t.Context()
	ctx = log.WithLogger(ctx, log.New(t.Output(), log.TextType, log.TraceLevel))

	sandboxPath := filepath.Join(t.TempDir(), "sandbox.json")
	t.Cleanup(func() {
		ctx := ctx
		if testCfg != nil {
			ctx = config.WithConfig(ctx, testCfg.Config)
		}

		if _, statErr := os.Stat(sandboxPath); os.IsNotExist(statErr) {
			return
		}

		sandbox, err := resource.LoadSandbox(sandboxPath, cmd.SandboxedResources...)
		require.NoError(t, err)
		require.NotNil(t, sandbox)

		require.NoError(t, sandbox.Teardown(context.WithoutCancel(ctx)))
	})

	return &testRunner{
		cfg: testCfg,
		TestEnv: &integ.TestEnv{
			UnikraftPath: ir.UnikraftPath,
			ConfigPath:   configPath,
			SandboxPath:  sandboxPath,
			Dir:          t.TempDir(),
		},
	}
}

type cliOpt func(*cliConfig)

type cliConfig struct {
	expectFail bool
	allowFail  bool
}

func expectFail() cliOpt {
	return func(c *cliConfig) {
		c.expectFail = true
	}
}

func allowFail() cliOpt {
	return func(c *cliConfig) {
		c.allowFail = true
	}
}

// cli runs a CLI command and returns the combined output.
func (r *testRunner) cli(t *testing.T, args []string, opts ...cliOpt) string {
	t.Helper()
	var cfg cliConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	out, err := r.RunCmd(t.Context(), t, args)
	switch {
	case cfg.expectFail:
		require.Error(t, err, "command %q was expected to fail but succeeded\n%s", strings.Join(args, " "), out)
	case cfg.allowFail:
		// ignore error
	default:
		require.NoError(t, err, "command %q failed\n%s", strings.Join(args, " "), out)
	}
	return out
}

// httpGet makes an HTTPS GET request with retries and returns the response body.
func (r *testRunner) httpGet(t *testing.T, url string) string {
	t.Helper()
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //#nosec G402 -- test code
		},
		Timeout: 10 * time.Second,
	}

	var lastErr error
	for range 10 {
		resp, err := client.Get(url) //#nosec G107 -- test code, URL from test
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			time.Sleep(2 * time.Second)
			continue
		}
		return string(body)
	}
	require.NoError(t, lastErr, "HTTP GET %s failed after retries", url)
	return ""
}
