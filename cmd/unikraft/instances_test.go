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

func instancesTestCases(t *testing.T, cfg *integration.Config) []testCase {
	t.Helper()
	if cfg == nil {
		t.Skip("integration config not found")
	}

	metroName := cfg.MetroName

	return []testCase{
		{
			name: "help",
			commands: []command{
				{args: []string{unikraftCmd, "instance", "--help"}},
				{args: []string{unikraftCmd, "instance", "get", "--help"}},
				{args: []string{unikraftCmd, "instance", "list", "--help"}},
				{args: []string{unikraftCmd, "instance", "wait", "--help"}},
				{args: []string{unikraftCmd, "instance", "create", "--help"}},
				{args: []string{unikraftCmd, "instance", "edit", "--help"}},
				{args: []string{unikraftCmd, "instance", "delete", "--help"}},
				{args: []string{unikraftCmd, "instance", "logs", "--help"}},
				{args: []string{unikraftCmd, "instance", "start", "--help"}},
				{args: []string{unikraftCmd, "instance", "stop", "--help"}},
				{args: []string{unikraftCmd, "instance", "restart", "--help"}},
			},
		},
		{
			name:   "create",
			online: true,
			commands: []command{
				{args: []string{unikraftCmd, "instance", "list"}},

				// Create an nginx instance
				{args: []string{
					unikraftCmd, "instance", "create",
					"--set", "name=test-$UNIQ_INST",
					"--set", "metro=" + metroName,
					"--set", "image=nginx:latest",
					"--set", "runtime.env=A=1,B=2,C=3",
					"--set", "autostart=true",
					"--set", "resources.memory=128",
					"--set", "resources.vcpus=1",
				}},

				{args: []string{unikraftCmd, "instance", "list"}},
				{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},

				{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "instance", "list"}},
			},
			cleaners: instanceCleaners,
		},
		{
			name:   "create-oom",
			online: true,
			commands: []command{
				{args: []string{
					unikraftCmd, "instance", "create",
					"--output", "quiet",
					"--set", "name=test-$UNIQ_INST",
					"--set", "metro=" + metroName,
					"--set", "image=nginx:latest",
					"--set", "autostart=true",
					"--set", "resources.memory=16Mib",
					"--set", "resources.vcpus=1",
				}},
				{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
			},
			cleaners: instanceCleaners,
		},
		{
			name:   "connect",
			online: true,
			commands: []command{
				// Create an nginx instance with a service
				{args: []string{
					unikraftCmd, "instance", "create",
					"--set", "name=test-$UNIQ_INST",
					"--set", "metro=" + metroName,
					"--set", "image=nginx:latest",
					"--set", "runtime.env=A=1,B=2,C=3",
					"--set", "autostart=true",
					"--set", "resources.memory=128",
					"--set", "resources.vcpus=1",
					"--set", "service.services=443:8080/tls+http",
					"--set", "service.domains=name=$UNIQ_DOMAIN",
				}},
				{
					args: []string{
						unikraftCmd, "instance", "inspect", "test-$UNIQ_INST",
						"--output", "template=" + `{{ (index .service.domains 0).fqdn }}`,
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
			},
			cleaners: instanceCleaners,
		},
		{
			name:   "start-stop",
			online: true,
			commands: []command{
				// Create an nginx instance
				{args: []string{
					unikraftCmd, "instance", "create",
					"--set", "name=test-$UNIQ_INST",
					"--set", "metro=" + metroName,
					"--set", "image=nginx:latest",
					"--set", "runtime.env=A=1,B=2,C=3",
					"--set", "autostart=true",
					"--set", "resources.memory=128",
					"--set", "resources.vcpus=1",
				}},

				{args: []string{unikraftCmd, "instance", "stop", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},

				{args: []string{unikraftCmd, "instance", "start", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},

				{args: []string{unikraftCmd, "instance", "edit", "test-$UNIQ_INST", "--set", "state=stopped"}},
				{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},

				{args: []string{unikraftCmd, "instance", "edit", "test-$UNIQ_INST", "--set", "state=running"}},
				{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},

				// {args: []string{unikraftCmd, "instance", "restart", "test-$UNIQ_INST"}},
				// {args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},

				{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
			},
			cleaners: instanceCleaners,
		},
		{
			name:   "edit",
			online: true,
			commands: []command{
				{args: []string{
					unikraftCmd, "instance", "create",
					"--output", "quiet",
					"--set", "name=test-$UNIQ_INST",
					"--set", "metro=" + metroName,
					"--set", "image=nginx:latest",
					"--set", "runtime.args=before,first",
					"--set", "runtime.env=A=1,B=2",
					"--set", "autostart=false",
					"--set", "resources.memory=128",
					"--set", "resources.vcpus=1",
				}},
				{args: []string{
					unikraftCmd, "instance", "edit", "test-$UNIQ_INST",
					"--output", "quiet",
					"--set", "image=redis:latest",
					"--set", "runtime.args=after,second",
					"--set", "runtime.env=A=3,B=4",
					"--set", "resources.memory=256",
					// "--set", "resources.vcpus=2",
				}},
				{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
			},
			cleaners: instanceCleaners,
		},
		{
			name:   "instances/volume",
			online: true,
			commands: []command{
				// Create a volume first
				{args: []string{
					unikraftCmd, "volume", "create",
					"--output", "quiet",
					"--set", "name=test-$UNIQ_VOL",
					"--set", "size=20",
					"--set", "metro=" + metroName,
				}},
				// Create an instance with the volume mounted at /mnt
				{args: []string{
					unikraftCmd, "instance", "create",
					"--set", "name=test-$UNIQ_INST",
					"--set", "metro=" + metroName,
					"--set", "image=nginx:latest",
					"--set", "autostart=true",
					"--set", "resources.memory=128",
					"--set", "resources.vcpus=1",
					"--set", "volumes=test-$UNIQ_VOL:/mnt",
				}},
				{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "volume", "delete", "test-$UNIQ_VOL"}},
			},
			cleaners: instanceCleaners,
		},
		{
			name:   "instances/volume-inline",
			online: true,
			commands: []command{
				// Create an instance with an inline volume (volume created automatically)
				// This tests the create-only "size" field in InstanceVolume
				// Format is :AT[:ro][:size=N] (no name, only size)
				{args: []string{
					unikraftCmd, "instance", "create",
					"--set", "name=test-$UNIQ_INST",
					"--set", "metro=" + metroName,
					"--set", "image=nginx:latest",
					"--set", "autostart=true",
					"--set", "resources.memory=128",
					"--set", "resources.vcpus=1",
					"--set", "volumes=:/data:size=20",
				}},
				{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
			},
			cleaners: instanceCleaners,
		},
		{
			name:   "instances/autostart",
			online: true,
			commands: []command{
				// Create an instance with autostart=true (should start automatically)
				{args: []string{
					unikraftCmd, "instance", "create",
					"--output", "quiet",
					"--set", "name=test-$UNIQ_INST",
					"--set", "metro=" + metroName,
					"--set", "image=nginx:latest",
					"--set", "autostart=true",
					"--set", "resources.memory=128",
					"--set", "resources.vcpus=1",
				}},
				// Verify instance is running (autostart worked)
				{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
			},
			cleaners: instanceCleaners,
		},
		{
			name:   "instances/add-domain",
			online: true,
			commands: []command{
				// Create an instance with a service (required to add domains later)
				{args: []string{
					unikraftCmd, "instance", "create",
					"--set", "name=test-$UNIQ_INST",
					"--set", "metro=" + metroName,
					"--set", "image=nginx:latest",
					"--set", "autostart=true",
					"--set", "resources.memory=128",
					"--set", "resources.vcpus=1",
					"--set", "service.services=443:8080/tls+http",
				}},
				// Capture the auto-generated service name
				{
					args: []string{
						unikraftCmd, "instance", "inspect", "test-$UNIQ_INST",
						"--output", "template={{ .service.name }}",
					},
					captureEnv: "SERVICE_NAME",
				},
				// Edit the service to add a domain
				{args: []string{
					unikraftCmd, "service", "edit", "$SERVICE_NAME",
					"--output", "quiet",
					"--add", "domains=name=$UNIQ_DOMAIN",
				}},
				// Verify instance now has the domain via the service
				{args: []string{unikraftCmd, "instance", "inspect", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
			},
			cleaners: instanceCleaners,
		},
	}
}

var instanceCleaners = []cleaner{
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
		// auto-generated volume names like "vol-0g8gc"
		pattern: regexp.MustCompile(`\bvol-[a-z0-9]+\b`),
		repl:    "<INLINE_VOL>",
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
	{
		// states can be stopping/stoped
		pattern: regexp.MustCompile(`\bstate:(\s+)(stopping|stopped)`),
		repl:    "state:${1}stopped",
	},
}
