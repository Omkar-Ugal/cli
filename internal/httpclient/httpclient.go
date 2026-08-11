// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package httpclient

import (
	"net/http"

	sdkhttpclient "unikraft.com/cloud/sdk/pkg/httpclient"

	"unikraft.com/cli/internal/version"
)

// GetClient returns an HTTP client based on the provided insecure flag.
func GetClient(insecure bool) *http.Client {
	if insecure {
		return InsecureHTTPClient
	}
	return DefaultHTTPClient
}

// DefaultHTTPClient is the default HTTP client used by the Unikraft CLI. It
// uses the environment's proxy settings and sets the CLI's User-Agent on
// requests that don't already set one.
var DefaultHTTPClient = sdkhttpclient.NewHTTPClient(
	sdkhttpclient.WithUserAgent(version.UserAgent()),
)

// InsecureHTTPClient is an HTTP client that skips TLS verification. It is
// intended for use in development or testing environments where self-signed
// certificates may be used.
var InsecureHTTPClient = sdkhttpclient.NewHTTPClient(
	sdkhttpclient.WithUserAgent(version.UserAgent()),
	sdkhttpclient.WithInsecure(),
)
