// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"regexp"
	"testing"

	"unikraft.com/cli/internal/integration"
)

func servicesTestCases(t *testing.T, cfg *integration.Config) []testCase {
	t.Helper()
	if cfg == nil {
		t.Skip("integration config not found")
	}

	metroName := cfg.MetroName

	return []testCase{
		{
			name: "help",
			commands: []command{
				{args: []string{unikraftCmd, "service", "--help"}},
				{args: []string{unikraftCmd, "service", "get", "--help"}},
				{args: []string{unikraftCmd, "service", "list", "--help"}},
				{args: []string{unikraftCmd, "service", "wait", "--help"}},
				{args: []string{unikraftCmd, "service", "create", "--help"}},
				{args: []string{unikraftCmd, "service", "edit", "--help"}},
				{args: []string{unikraftCmd, "service", "delete", "--help"}},
			},
		},
		{
			name:   "create",
			online: true,
			commands: []command{
				{args: []string{unikraftCmd, "service", "list"}},
				{args: []string{
					unikraftCmd, "service", "create",
					"--set", "name=test-$UNIQ_SVC_A",
					"--set", "metro=" + metroName,
					"--set", "domains=fqdn=$UNIQ_DOMAIN_A.unikraft.example",
					"--set", "services=443:8080/tls+http",
					"--set", "services=80:443/http+redirect",
				}},
				{args: []string{
					unikraftCmd, "service", "create",
					"--set", "name=test-$UNIQ_SVC_B",
					"--set", "metro=" + metroName,
					"--set", "domains=fqdn=$UNIQ_DOMAIN_B.unikraft.example",
					"--set", "services=443:8080/tls+http,80:443/http+redirect",
				}},
				{args: []string{unikraftCmd, "service", "list"}},
				{args: []string{unikraftCmd, "service", "inspect", "test-$UNIQ_SVC_A", "test-$UNIQ_SVC_B"}},

				{args: []string{unikraftCmd, "service", "delete", "test-$UNIQ_SVC_A", "test-$UNIQ_SVC_B"}},
			},
			cleaners: serviceCleaners,
		},
		{
			name:   "edit",
			online: true,
			commands: []command{
				{args: []string{
					unikraftCmd, "service", "create",
					"--output", "quiet",
					"--set", "name=test-$UNIQ_SVC",
					"--set", "metro=" + metroName,
					"--set", "domains=fqdn=$UNIQ_DOMAIN.unikraft.example",
					"--set", "services=443:8080/tls+http",
				}},
				{args: []string{
					unikraftCmd, "service", "edit", "test-$UNIQ_SVC",
					"--output", "quiet",
					"--set", "limits.soft=2",
					"--set", "limits.hard=10",
					"--set", "domains=fqdn=$UNIQ_DOMAIN_EDIT.unikraft.example",
					"--set", "services=1000:2000/tls",
				}},
				{args: []string{unikraftCmd, "service", "inspect", "test-$UNIQ_SVC"}},
				{args: []string{unikraftCmd, "service", "delete", "test-$UNIQ_SVC"}},
			},
			cleaners: serviceCleaners,
		},
	}
}

var serviceCleaners = []cleaner{
	{
		// automatically generated certificate names
		pattern: regexp.MustCompile(`\.unikraft\.example-[a-z0-9]{5,}`),
		repl:    ".unikraft.example-xxxxx",
	},
}
