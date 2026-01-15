// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

//go:build integration

package main

var helpTestCases = []testCase{
	{
		name:     "empty",
		commands: []command{{args: []string{unikraftCmd}, allowErr: true}},
	},
	{
		name:     "help",
		commands: []command{{args: []string{unikraftCmd, "--help"}}},
	},
	{
		name:     "version",
		commands: []command{{args: []string{unikraftCmd, "--version"}}},
	},
}
