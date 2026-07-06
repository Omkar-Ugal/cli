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

// SkipUnlessSupportedMetroVersion skips the test when the metro version is
// not supported.
func SkipUnlessSupportedMetroVersion(t testing.TB, metros []string) {
	t.Helper()
	metro := os.Getenv("CLI_TEST_SERVER")
	if metro == "" {
		return
	}

	if slices.Contains(metros, metro) {
		return
	}

	t.Skip("skipping test (unsupported metro version)")
}
