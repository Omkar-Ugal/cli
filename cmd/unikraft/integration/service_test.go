// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServices(t *testing.T) {
	ir := newIntegrationRunner(t)

	t.Run("create", func(t *testing.T) {
		r := ir.runner(t, true)
		svcNameA := uniq()
		svcNameB := uniq()
		domainA := uniq()
		domainB := uniq()

		out := r.cli(t, []string{"unikraft", "service", "list"})
		assert.Regexp(t, `METRO\s+NAME`, out)

		out = r.cli(t, []string{
			"unikraft", "service", "create",
			"--set", "name=test-" + svcNameA,
			"--set", "metro=" + ir.cfg.MetroName,
			"--set", "domains=fqdn=" + domainA + ".unikraft.example",
			"--set", "services=443:8080/tls+http",
			"--set", "services=80:443/http+redirect",
		})
		assert.Regexp(t, `name:\s+test-`, out)
		assert.Regexp(t, `source:\s+443`, out)
		assert.Regexp(t, `destination:\s+8080`, out)
		assert.Regexp(t, `source:\s+80`, out)
		assert.Regexp(t, `destination:\s+443`, out)
		assert.Regexp(t, `fqdn:`, out)

		out = r.cli(t, []string{
			"unikraft", "service", "create",
			"--set", "name=test-" + svcNameB,
			"--set", "metro=" + ir.cfg.MetroName,
			"--set", "domains=fqdn=" + domainB + ".unikraft.example",
			"--set", "services=443:8080/tls+http",
			"--set", "services=80:443/http+redirect",
		})
		assert.Regexp(t, `name:\s+test-`, out)
		assert.Regexp(t, `source:\s+443`, out)
		assert.Regexp(t, `destination:\s+8080`, out)
		assert.Regexp(t, `source:\s+80`, out)
		assert.Regexp(t, `destination:\s+443`, out)
		assert.Regexp(t, `fqdn:`, out)

		out = r.cli(t, []string{"unikraft", "service", "list"})
		assert.Regexp(t, `test-`, out)

		out = r.cli(t, []string{"unikraft", "service", "inspect", "test-" + svcNameA, "test-" + svcNameB})
		assert.Regexp(t, `source:\s+443`, out)
		assert.Regexp(t, `destination:\s+8080`, out)
		assert.Regexp(t, `source:\s+80`, out)
		assert.Regexp(t, `destination:\s+443`, out)
		assert.Regexp(t, `fqdn:`, out)

		out = r.cli(t, []string{"unikraft", "service", "delete", "test-" + svcNameA, "test-" + svcNameB})
		assert.Regexp(t, `test-`, out)
	})

	t.Run("edit", func(t *testing.T) {
		r := ir.runner(t, true)
		svcName := uniq()
		domainName := uniq()
		domainEdit := uniq()

		r.cli(t, []string{
			"unikraft", "service", "create",
			"--output", "quiet",
			"--set", "name=test-" + svcName,
			"--set", "metro=" + ir.cfg.MetroName,
			"--set", "domains=fqdn=" + domainName + ".unikraft.example",
			"--set", "services=443:8080/tls+http",
		})

		r.cli(t, []string{
			"unikraft", "service", "edit", "test-" + svcName,
			"--output", "quiet",
			"--set", "limits.soft=2",
			"--set", "limits.hard=10",
			"--set", "domains=fqdn=" + domainEdit + ".unikraft.example",
			"--set", "services=1000:2000/tls",
		})

		out := r.cli(t, []string{"unikraft", "service", "inspect", "test-" + svcName})
		assert.Regexp(t, `soft:\s+2`, out)
		assert.Regexp(t, `hard:\s+10`, out)
		assert.Regexp(t, `source:\s+1000`, out)
		assert.Regexp(t, `destination:\s+2000`, out)
		assert.Regexp(t, `fqdn:`, out)

		r.cli(t, []string{"unikraft", "service", "delete", "test-" + svcName})
	})
}
