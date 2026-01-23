// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

var certificatesTestCases = []testCase{
	{
		name: "certificates/help",
		commands: []command{
			{args: []string{unikraftCmd, "certificate", "--help"}},
			{args: []string{unikraftCmd, "certificate", "get", "--help"}},
			{args: []string{unikraftCmd, "certificate", "list", "--help"}},
			{args: []string{unikraftCmd, "certificate", "wait", "--help"}},
			{args: []string{unikraftCmd, "certificate", "create", "--help"}},
			{args: []string{unikraftCmd, "certificate", "delete", "--help"}},
		},
	},
	{
		name:   "certificates/create",
		online: true,
		commands: []command{
			{args: []string{unikraftCmd, "certificate", "list"}},
			{args: []string{unikraftCmd, "certificate", "create", "--set", "name=test-$UNIQ_CERT_A", "--set", "cn=$CERT_A_CN", "--set", "chain=$CERT_A_CHAIN", "--set", "pkey=$CERT_A_KEY", "--set", "metro=" + defaultMetro}},
			{args: []string{unikraftCmd, "certificate", "create", "--set", "name=test-$UNIQ_CERT_B", "--set", "cn=$CERT_B_CN", "--set", "chain=$CERT_B_CHAIN", "--set", "pkey=$CERT_B_KEY", "--set", "metro=" + defaultMetro}},
			{args: []string{unikraftCmd, "certificate", "list"}},
			{args: []string{unikraftCmd, "certificate", "inspect", "test-$UNIQ_CERT_A", "test-$UNIQ_CERT_B"}},
			{args: []string{unikraftCmd, "certificate", "delete", "test-$UNIQ_CERT_A", "test-$UNIQ_CERT_B"}},
		},
	},
}
