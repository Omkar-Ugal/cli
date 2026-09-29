// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdkhttpclient "unikraft.com/cloud/sdk/pkg/httpclient"
)

// TestRegistryClientHasNoRequestTimeout pins the regression that broke pushing
// a multi-gigabyte rootfs: http.Client.Timeout covers writing the request body,
// so any non-zero value kills a large blob PUT mid-upload.
func TestRegistryClientHasNoRequestTimeout(t *testing.T) {
	assert.Zero(t, RegistryHTTPClient.Timeout)
	assert.Zero(t, InsecureRegistryHTTPClient.Timeout)
}

// TestResponseHeaderTimeoutReachesClient pins the SDK behaviour the registry
// clients rely on: WithResponseHeaderTimeout applies to the transport that
// serves requests.
func TestResponseHeaderTimeoutReachesClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	_, err := sdkhttpclient.NewHTTPClient(
		sdkhttpclient.WithUserAgent("test"),
		sdkhttpclient.WithResponseHeaderTimeout(20*time.Millisecond),
	).Get(server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout awaiting response headers")
}
