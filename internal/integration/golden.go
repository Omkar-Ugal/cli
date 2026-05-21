// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"strings"
	"testing"

	"gotest.tools/v3/golden"
)

// Gild runs a callback for each arg, concatenates the outputs, and asserts
// against the golden file for the current test. Only use offline callbacks.
func Gild[Arg any](t *testing.T, callback func(*testing.T, Arg) string, args ...Arg) {
	t.Helper()
	var output strings.Builder
	for i, arg := range args {
		if i > 0 {
			output.WriteString("\n")
		}
		output.WriteString(callback(t, arg))
	}
	golden.Assert(t, output.String(), t.Name())
}
