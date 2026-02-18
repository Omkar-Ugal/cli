// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").

package main

var resourceTestCases = []testCase{
	{
		name: "resource/help",
		commands: []command{
			{args: []string{unikraftCmd, "resource", "--help"}},
		},
	},
	{
		name: "resource/purge/help",
		commands: []command{
			{args: []string{unikraftCmd, "resource", "purge", "--help"}},
		},
	},
	{
		name:   "resource/volume-flow",
		online: true,
		commands: []command{
			{args: []string{unikraftCmd, "resource", "create", "--set", "type=volume", "--set", "name=test-$UNIQ_VOLUME", "--set", "size=10", "--set", "metro=" + defaultMetro}},
			{args: []string{unikraftCmd, "resource", "get", "volume:" + defaultMetro + "/test-$UNIQ_VOLUME"}},
			{args: []string{unikraftCmd, "resource", "list"}},
			{args: []string{unikraftCmd, "resource", "edit", "volume:" + defaultMetro + "/test-$UNIQ_VOLUME", "--set", "size=20"}},
			{args: []string{unikraftCmd, "resource", "get", "volume:" + defaultMetro + "/test-$UNIQ_VOLUME"}},
			{args: []string{unikraftCmd, "volume", "get", "test-$UNIQ_VOLUME"}},
			{args: []string{unikraftCmd, "resource", "purge", "--force"}},
			{args: []string{unikraftCmd, "volume", "ls"}},
		},
	},
}
