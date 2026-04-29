// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"regexp"
	"testing"
)

func volumesTests(t *testing.T, r *testRunner) {
	t.Run("help", func(t *testing.T) {
		r.run(t, []command{
			{args: []string{unikraftCmd, "volume", "--help"}},
			{args: []string{unikraftCmd, "volume", "get", "--help"}},
			{args: []string{unikraftCmd, "volume", "list", "--help"}},
			{args: []string{unikraftCmd, "volume", "wait", "--help"}},
			{args: []string{unikraftCmd, "volume", "create", "--help"}},
			{args: []string{unikraftCmd, "volume", "clone", "--help"}},
			{args: []string{unikraftCmd, "volume", "import", "--help"}},
			{args: []string{unikraftCmd, "volume", "edit", "--help"}},
			{args: []string{unikraftCmd, "volume", "delete", "--help"}},
		})
	})

	metroName := ""
	if r.cfg != nil {
		metroName = r.cfg.MetroName
	}

	t.Run("create", func(t *testing.T) {
		r.
			online().
			run(t, []command{
				{args: []string{unikraftCmd, "volume", "list"}},
				{args: []string{unikraftCmd, "volume", "create", "--set", "name=test-$UNIQ_VOLUME", "--set", "size=10", "--set", "metro=" + metroName}},
				{args: []string{unikraftCmd, "volume", "list"}},
				{args: []string{unikraftCmd, "volume", "inspect", "test-$UNIQ_VOLUME"}},
				{args: []string{unikraftCmd, "volume", "delete", "test-$UNIQ_VOLUME"}},
			})
	})

	t.Run("edit", func(t *testing.T) {
		r.
			online().
			run(t, []command{
				{args: []string{unikraftCmd, "volume", "create", "--output", "quiet", "--set", "name=test-$UNIQ_VOLUME", "--set", "size=10", "--set", "metro=" + metroName}},
				{args: []string{unikraftCmd, "volume", "edit", "test-$UNIQ_VOLUME", "--output", "quiet", "--set", "size=20"}},
				{args: []string{unikraftCmd, "volume", "inspect", "test-$UNIQ_VOLUME"}},
				{args: []string{unikraftCmd, "volume", "delete", "test-$UNIQ_VOLUME"}},
			})
	})

	t.Run("clone", func(t *testing.T) {
		r.
			online().
			run(t, []command{
				{args: []string{unikraftCmd, "volume", "create", "--output", "quiet", "--set", "name=test-$UNIQ_VOLUME", "--set", "size=10", "--set", "metro=" + metroName}},
				{args: []string{unikraftCmd, "volume", "clone", "test-$UNIQ_VOLUME", "--output", "quiet", "--set", "name=test-$UNIQ_VOLUME_CLONE"}},
				{args: []string{unikraftCmd, "volume", "inspect", "test-$UNIQ_VOLUME", "test-$UNIQ_VOLUME_CLONE"}},
				{args: []string{unikraftCmd, "volume", "delete", "test-$UNIQ_VOLUME", "test-$UNIQ_VOLUME_CLONE"}},
			})
	})

	t.Run("import", func(t *testing.T) {
		// Offline: missing --source errors before any network call.
		t.Run("missing-source", func(t *testing.T) {
			r.run(t, []command{
				{args: []string{unikraftCmd, "volume", "import", "my-volume"}, err: errYes},
			})
		})

		// Offline: port below the allowed range errors before any network call.
		t.Run("invalid-port", func(t *testing.T) {
			r.run(t, []command{
				{args: []string{unikraftCmd, "volume", "import", "my-volume", "--source", ".", "--port", "80"}, err: errYes},
			})
		})

		// Offline: port above the allowed range errors before any network call.
		t.Run("invalid-port-high", func(t *testing.T) {
			r.run(t, []command{
				{args: []string{unikraftCmd, "volume", "import", "my-volume", "--source", ".", "--port", "99999"}, err: errYes},
			})
		})

		// Import a small directory into a freshly created volume.
		t.Run("dir", func(t *testing.T) {
			r.
				online().
				withCleaners([]cleaner{
					// free size differs on every run
					{
						pattern: regexp.MustCompile(`free=[0-9]+.?[0-9]*MiB`),
						repl:    "free=10MiB",
					},
				}).
				withContext(map[string]string{
					"hello.txt": "hello from volume import\n",
				}).
				run(t, []command{
					{args: []string{unikraftCmd, "volume", "create", "--output", "quiet", "--set", "name=test-$UNIQ_VOLUME", "--set", "size=10", "--set", "metro=" + metroName}},
					{args: []string{unikraftCmd, "volume", "import", "test-$UNIQ_VOLUME", "--source", "."}},
					{args: []string{unikraftCmd, "volume", "inspect", "test-$UNIQ_VOLUME"}},
					{args: []string{unikraftCmd, "volume", "delete", "test-$UNIQ_VOLUME"}},
				})
		})

		// Import a file into a freshly created volume and test connection.
		t.Run("serve", func(t *testing.T) {
			r.
				online().
				withCleaners(instanceCleaners).
				withCleaners([]cleaner{
					{
						// free size differs on every run
						pattern: regexp.MustCompile(`free=[0-9]+.?[0-9]*MiB`),
						repl:    "free=50MiB",
					},
				}).
				withContext(map[string]string{
					"index.html": "<html><body>hello from volume import</body></html>\n",
				}).
				run(t, []command{
					// Create a volume to hold the custom web content
					{args: []string{
						unikraftCmd, "volume", "create",
						"--output", "quiet",
						"--set", "name=test-$UNIQ_VOL",
						"--set", "size=50",
						"--set", "metro=" + metroName,
					}},
					// Import the custom index.html into the volume
					{args: []string{unikraftCmd, "volume", "import", "test-$UNIQ_VOL", "--source", "."}},
					{args: []string{
						unikraftCmd, "instance", "create",
						"--set", "name=test-$UNIQ_INST",
						"--set", "metro=" + metroName,
						"--set", "image=nginx:latest",
						"--set", "autostart=true",
						"--set", "resources.memory=256",
						"--set", "resources.vcpus=1",
						"--set", "volumes=test-$UNIQ_VOL:/wwwroot",
						"--set", "service.services=443:8080/tls+http",
						"--set", "service.domains=name=$UNIQ_DOMAIN",
					}},
					// Capture the assigned FQDN
					{
						args: []string{
							unikraftCmd, "instance", "inspect", "test-$UNIQ_INST",
							"--output", "template=" + `{{ (index .service.domains 0).fqdn }}`,
						},
						captureEnv: "FQDN",
					},
					{args: []string{unikraftCmd, "instance", "wait", "--until", "state==running", "--timeout", "30s", "test-$UNIQ_INST"}},
					// Curl the instance and write the body to a file for content verification.
					{args: []string{
						"curl",
						"-k",
						"--fail",
						"--silent",
						"--show-error",
						"--output", "response.html",
						"--retry", "10",
						"--retry-delay", "2",
						"--retry-all-errors",
						"--connect-timeout", "5",
						"--max-time", "10",
						"https://$FQDN",
					}},
					// Assert the imported content is served.
					{args: []string{"grep", "hello from volume import", "response.html"}},
					{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
					{args: []string{unikraftCmd, "volume", "delete", "test-$UNIQ_VOL"}},
				})
		})
	})
}
