// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"os"
	"slices"
	"testing"
)

// SkipUnlessIntegration skips the test when the integration tag is not set.
func SkipUnlessIntegration(t testing.TB) {
	t.Helper()
	if !integrationEnabled {
		t.Skip("skipping integration test (missing integration build tag)")
	}
}

// SkipUnlessSupportedServer skips the test when the server is
// not supported.
func SkipUnlessSupportedServer(t testing.TB, servers []string) {
	t.Helper()
	server := os.Getenv("UNIKRAFT_X_TEST_SERVER")
	if server == "" {
		return
	}

	if slices.Contains(servers, server) {
		return
	}

	t.Skip("skipping test (unsupported server)")
}
