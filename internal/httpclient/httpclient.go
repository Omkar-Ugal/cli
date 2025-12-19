// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package httpclient

import (
	"crypto/tls"
	"net/http"
	"net/url"

	"golang.org/x/net/http/httpproxy"

	"unikraft.com/cli/internal/version"
)

// DefaultHTTPClient is the default HTTP client used by the Unikraft CLI.  It
// sets a custom User-Agent header and uses the environment's proxy settings,
// which allows for easy debugging.
var DefaultHTTPClient = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
		Proxy: func(req *http.Request) (*url.URL, error) {
			req.Header.Set("User-Agent", version.UserAgent())
			return httpproxy.FromEnvironment().ProxyFunc()(req.URL)
		},
	},
}

// InsecureHTTPClient is an HTTP client that skips TLS verification. It is
// intended for use in development or testing environments where self-signed
// certificates may be used.
var InsecureHTTPClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
		Proxy: func(req *http.Request) (*url.URL, error) {
			req.Header.Set("User-Agent", version.UserAgent())
			return httpproxy.FromEnvironment().ProxyFunc()(req.URL)
		},
	},
}
