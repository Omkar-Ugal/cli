// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"strings"
	"testing"

	"github.com/containerd/continuity/fs/fstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVolumes(t *testing.T) {
	ir := newIntegrationRunner(t)

	t.Run("create", func(t *testing.T) {
		r := ir.runner(t, true)
		volName := uniq()

		out := r.cli(t, []string{"unikraft", "volume", "list"})
		assert.Regexp(t, `METRO\s+NAME`, out)

		out = r.cli(t, []string{"unikraft", "volume", "create", "--set", "name=test-" + volName, "--set", "size=10", "--set", "metro=" + ir.cfg.MetroName})
		assert.Regexp(t, `state:\s+available`, out)
		assert.Regexp(t, `size:\s+10`, out)
		assert.Regexp(t, `filesystem:`, out)

		out = r.cli(t, []string{"unikraft", "volume", "list"})
		assert.Regexp(t, `test-.*available`, out)

		out = r.cli(t, []string{"unikraft", "volume", "inspect", "test-" + volName})
		assert.Regexp(t, `state:\s+available`, out)
		assert.Regexp(t, `size:\s+10`, out)

		out = r.cli(t, []string{"unikraft", "volume", "delete", "test-" + volName})
		assert.Regexp(t, `test-`, out)
	})

	t.Run("edit", func(t *testing.T) {
		r := ir.runner(t, true)
		volName := uniq()

		r.cli(t, []string{"unikraft", "volume", "create", "--output", "quiet", "--set", "name=test-" + volName, "--set", "size=10", "--set", "metro=" + ir.cfg.MetroName})
		r.cli(t, []string{"unikraft", "volume", "edit", "test-" + volName, "--output", "quiet", "--set", "size=20"})

		out := r.cli(t, []string{"unikraft", "volume", "inspect", "test-" + volName})
		assert.Regexp(t, `size:\s+20`, out)

		r.cli(t, []string{"unikraft", "volume", "delete", "test-" + volName})
	})

	t.Run("clone", func(t *testing.T) {
		r := ir.runner(t, true)
		volName := uniq()
		cloneName := uniq()

		r.cli(t, []string{"unikraft", "volume", "create", "--output", "quiet", "--set", "name=test-" + volName, "--set", "size=10", "--set", "metro=" + ir.cfg.MetroName})
		r.cli(t, []string{"unikraft", "volume", "clone", "test-" + volName, "--output", "quiet", "--set", "name=test-" + cloneName})

		out := r.cli(t, []string{"unikraft", "volume", "inspect", "test-" + volName, "test-" + cloneName})
		assert.Regexp(t, `state:\s+available`, out)
		assert.Regexp(t, `size:\s+10`, out)

		r.cli(t, []string{"unikraft", "volume", "delete", "test-" + volName})
		r.cli(t, []string{"unikraft", "volume", "delete", "test-" + cloneName})
	})

	t.Run("import", func(t *testing.T) {
		t.Run("missing-source", func(t *testing.T) {
			r := ir.runner(t, false)
			out := r.cli(t, []string{"unikraft", "volume", "import", "my-volume"}, expectFail())
			assert.Regexp(t, `source path is required`, out)
		})

		t.Run("invalid-port", func(t *testing.T) {
			r := ir.runner(t, false)
			out := r.cli(t, []string{"unikraft", "volume", "import", "my-volume", "--source", ".", "--port", "80"}, expectFail())
			assert.Regexp(t, `port must be between`, out)
		})

		t.Run("invalid-port-high", func(t *testing.T) {
			r := ir.runner(t, false)
			out := r.cli(t, []string{"unikraft", "volume", "import", "my-volume", "--source", ".", "--port", "99999"}, expectFail())
			assert.Regexp(t, `port must be between`, out)
		})

		t.Run("dir", func(t *testing.T) {
			r := ir.runner(t, true)
			volName := uniq()
			require.NoError(t, fstest.Apply(
				fstest.CreateFile("hello.txt", []byte("hello from volume import\n"), 0o644),
			).Apply(r.Dir))

			r.cli(t, []string{"unikraft", "volume", "create", "--output", "quiet", "--set", "name=test-" + volName, "--set", "size=10", "--set", "metro=" + ir.cfg.MetroName})

			out := r.cli(t, []string{"unikraft", "volume", "import", "test-" + volName, "--source", "."})
			assert.Regexp(t, `import complete`, out)

			out = r.cli(t, []string{"unikraft", "volume", "inspect", "test-" + volName})
			assert.Regexp(t, `state:\s+available`, out)

			r.cli(t, []string{"unikraft", "volume", "delete", "test-" + volName})
		})

		t.Run("serve", func(t *testing.T) {
			r := ir.runner(t, true)
			volName := uniq()
			instName := uniq()
			domainName := uniq()
			require.NoError(t, fstest.Apply(
				fstest.CreateFile("index.html", []byte("<html><body>hello from volume import</body></html>\n"), 0o644),
			).Apply(r.Dir))

			r.cli(t, []string{
				"unikraft", "volume", "create",
				"--output", "quiet",
				"--set", "name=test-" + volName,
				"--set", "size=50",
				"--set", "metro=" + ir.cfg.MetroName,
			})

			out := r.cli(t, []string{"unikraft", "volume", "import", "test-" + volName, "--source", "."})
			assert.Regexp(t, `import complete`, out)

			r.cli(t, []string{
				"unikraft", "instance", "create",
				"--set", "name=test-" + instName,
				"--set", "metro=" + ir.cfg.MetroName,
				"--set", "image=nginx:latest",
				"--set", "autostart=true",
				"--set", "resources.memory=256",
				"--set", "resources.vcpus=1",
				"--set", "volumes=test-" + volName + ":/wwwroot",
				"--set", "service.services=443:8080/tls+http",
				"--set", "service.domains=name=" + domainName,
			})

			out = r.cli(t, []string{
				"unikraft", "instance", "inspect", "test-" + instName,
				"--output", "template=" + `{{ (index .service.domains 0).fqdn }}`,
			})
			fqdn := strings.TrimSpace(out)

			r.cli(t, []string{"unikraft", "instance", "wait", "--until", "state==running", "--timeout", "30s", "test-" + instName})

			body := r.httpGet(t, "https://"+fqdn)
			assert.Contains(t, body, "hello from volume import")

			r.cli(t, []string{"unikraft", "instance", "delete", "test-" + instName})
			r.cli(t, []string{"unikraft", "volume", "delete", "test-" + volName})
		})
	})
}
