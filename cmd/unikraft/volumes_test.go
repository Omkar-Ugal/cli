// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

var volumesTestCases = []testCase{
	{
		name: "volumes/help",
		commands: []command{
			{args: []string{unikraftCmd, "volume", "--help"}},
			{args: []string{unikraftCmd, "volume", "get", "--help"}},
			{args: []string{unikraftCmd, "volume", "list", "--help"}},
			{args: []string{unikraftCmd, "volume", "wait", "--help"}},
			{args: []string{unikraftCmd, "volume", "create", "--help"}},
			{args: []string{unikraftCmd, "volume", "edit", "--help"}},
			{args: []string{unikraftCmd, "volume", "delete", "--help"}},
		},
	},
	{
		name:   "volumes/create",
		online: true,
		commands: []command{
			{args: []string{unikraftCmd, "volume", "list"}},
			{args: []string{unikraftCmd, "volume", "create", "--set", "name=test-$UNIQ_VOLUME", "--set", "size=10", "--set", "metro=" + defaultMetro}},
			{args: []string{unikraftCmd, "volume", "list"}},
			{args: []string{unikraftCmd, "volume", "inspect", "test-$UNIQ_VOLUME"}},
			{args: []string{unikraftCmd, "volume", "delete", "test-$UNIQ_VOLUME"}},
		},
	},
	{
		name:   "volumes/edit",
		online: true,
		commands: []command{
			{args: []string{unikraftCmd, "volume", "create", "--output", "quiet", "--set", "name=test-$UNIQ_VOLUME", "--set", "size=10", "--set", "metro=" + defaultMetro}},
			{args: []string{unikraftCmd, "volume", "edit", "test-$UNIQ_VOLUME", "--output", "quiet", "--set", "size=20"}},
			{args: []string{unikraftCmd, "volume", "inspect", "test-$UNIQ_VOLUME"}},
			{args: []string{unikraftCmd, "volume", "delete", "test-$UNIQ_VOLUME"}},
		},
	},
}
