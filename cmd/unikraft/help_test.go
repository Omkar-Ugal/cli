// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"testing"

	"unikraft.com/cli/internal/integration"
)

func helpTestCases(t *testing.T, _ *integration.Config) []testCase {
	t.Helper()

	return []testCase{
		{
			name:     "empty",
			commands: []command{{args: []string{unikraftCmd}, allowErr: true}},
		},
		{
			name:     "version",
			commands: []command{{args: []string{unikraftCmd, "--version"}}},
		},
		{
			name:     "help",
			commands: []command{{args: []string{unikraftCmd, "--help"}}},
		},
		{
			// check we can detect an invalid arg
			name:     "invalid/arg",
			commands: []command{{args: []string{unikraftCmd, "invalid"}, allowErr: true}},
		},
		{
			// check help args still work when we have an invalid cmdline
			name: "invalid/help",
			commands: []command{
				{args: []string{unikraftCmd, "--help", "--bad-flag"}, allowErr: true},
				{args: []string{unikraftCmd, "--help", "bad-arg"}, allowErr: true},
				// NOTE: these aren't handled, since parsing exits *immediately* after
				// finding an invalid flag/arg
				// {args: []string{unikraftCmd, "--bad-flag", "--help"}, allowErr: true},
				// {args: []string{unikraftCmd, "bad-arg", "--help"}, allowErr: true},
			},
		},
		{
			// check log args still work when we have an invalid cmdline
			name: "invalid/logs",
			commands: []command{
				{args: []string{unikraftCmd, "--log-type=json", "invalid"}, allowErr: true},
				{args: []string{unikraftCmd, "--log-level=fatal", "invalid"}, allowErr: true},
			},
		},
	}
}
