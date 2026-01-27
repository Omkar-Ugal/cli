// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import "regexp"

var imagesTestCases = []testCase{
	{
		name: "images/help",
		commands: []command{
			{args: []string{unikraftCmd, "image", "--help"}},
			{args: []string{unikraftCmd, "image", "get", "--help"}},
			{args: []string{unikraftCmd, "image", "list", "--help"}},
			{args: []string{unikraftCmd, "image", "wait", "--help"}},
			{args: []string{unikraftCmd, "image", "copy", "--help"}},
		},
	},
	{
		name:   "images/inspect",
		online: true,
		commands: []command{
			{args: []string{unikraftCmd, "image", "list", "--filter", `ref~="/official/nginx:latest$"`}},
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
