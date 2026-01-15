// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

//go:build integration

package main

var authTestCases = []testCase{
	{
		name:   "auth",
		online: true,
		commands: []command{
			{args: []string{unikraftCmd, "login"}, token: true},
			{args: []string{unikraftCmd, "profile", "list"}},
			{args: []string{unikraftCmd, "metro", "list"}},
			{args: []string{unikraftCmd, "logout"}},
		},
	},
}
