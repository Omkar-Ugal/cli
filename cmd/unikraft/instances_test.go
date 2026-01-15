// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

//go:build integration

package main

import "regexp"

var instancesTestCases = []testCase{
	{
		name:   "instances",
		online: true,
		commands: []command{
			{args: []string{unikraftCmd, "login"}, token: true},
			{args: []string{unikraftCmd, "instance", "list"}},

			// Create an nginx instance with a service
			{args: []string{
				unikraftCmd, "instance", "create",
				"--set", "name=test-$UNIQ_INST",
				"--set", "metro=" + defaultMetro,
				"--set", "image=nginx:latest",
				"--set", "autostart=true",
				"--set", "resources.memory=128",
				"--set", "resources.vcpus=1",
				"--set", "service.services=443:8080/tls+http",
				"--set", "service.domains=name=$UNIQ_DOMAIN",
			}},

			// HACK: sleep to allow for instance to be ready
			{args: []string{"sleep", "5"}},

			{args: []string{unikraftCmd, "instance", "list"}},
			{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},

			// {args: []string{unikraftCmd, "instance", "logs", "test-$UNIQ_INST"}},

			{args: []string{unikraftCmd, "instance", "stop", "test-$UNIQ_INST"}},
			{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},

			{args: []string{unikraftCmd, "instance", "start", "test-$UNIQ_INST"}},
			{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},
			{
				args: []string{
					unikraftCmd, "instance", "inspect", "test-$UNIQ_INST",
					"--format", `{{ (index (index (index (index . 0) "service") "domains") 0).fqdn }}`,
				},
				captureEnv: "FQDN",
			},

			{args: []string{
				"curl",
				"-k",
				"--fail",
				"--silent",
				"--show-error",
				"--output", "/dev/null",
				"--write-out", `HTTP %{http_code} OK\n%header{server}\n`,
				"--retry", "10",
				"--retry-delay", "2",
				"--retry-all-errors",
				"--connect-timeout", "5",
				"--max-time", "10",
				"https://$FQDN",
			}},

			{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
			{args: []string{unikraftCmd, "instance", "list"}},
		},
		cleaners: []cleaner{
			{
				// auto-generated service names like "falling-sky-7cay704w"
				pattern: regexp.MustCompile(`\b[a-z]+-[a-z]+-[a-z0-9]{8}\b`),
				repl:    "<SERVICE_NAME>",
			},
			{
				// auto-generated domain names like "foo.ukp-stable.apw.unikraft.internal"
				pattern: regexp.MustCompile(`\b\.[a-z0-9.\-]+\.unikraft\.(app|internal)\b`),
				repl:    ".unikraft.internal",
			},
			{
				// IP addresses like "10.0.1.29"
				pattern: regexp.MustCompile(`\b10\.\d+\.\d+\.\d+\b`),
				repl:    "10.X.X.X",
			},
			{
				// MAC addresses like "12:b0:0a:HH:MM:1d" (already partially cleaned)
				pattern: regexp.MustCompile(`[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}`),
				repl:    "aa:bb:cc:dd:ee:ff",
			},
			{
				// states can be running/starting
				pattern: regexp.MustCompile(`\bstate:(\s+)(running|starting)`),
				repl:    "state:${1}running",
			},
		},
	},
}
