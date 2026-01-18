// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

//go:build integration

package main

import "regexp"

var imagesTestCases = []testCase{
	{
		name:   "images",
		online: true,
		commands: []command{
			{args: []string{unikraftCmd, "login"}, token: true},
			{args: []string{unikraftCmd, "image", "list", "--filter", "ref~=^nginx:"}},
			{args: []string{unikraftCmd, "image", "inspect", "nginx:latest"}},
		},
		cleaners: []cleaner{
			{
				// exact nginx version numbers may change between runs
				pattern: regexp.MustCompile(`nginx:[0-9]+\.[0-9]+`),
				repl:    "nginx:X.Y",
			},
		},
	},
}
