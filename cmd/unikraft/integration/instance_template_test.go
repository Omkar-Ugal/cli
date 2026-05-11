// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceTemplates(t *testing.T) {
	ir := newIntegrationRunner(t)

	t.Run("template", func(t *testing.T) {
		r := ir.runner(t, true)
		instName := uniq()

		r.cli(t, []string{
			"unikraft", "instance", "create",
			"--output", "quiet",
			"--set", "name=test-" + instName,
			"--set", "metro=" + ir.cfg.MetroName,
			"--set", "image=nginx:latest",
			"--set", "autostart=false",
			"--set", "resources.memory=128",
			"--set", "resources.vcpus=1",
		})

		out := r.cli(t, []string{
			"unikraft", "instance", "template", "create", "test-" + instName,
			"--output", "template={{ .name }}",
		})
		templateName := strings.TrimSpace(out)

		out = r.cli(t, []string{"unikraft", "instance", "template", "list"})
		assert.Regexp(t, `NAME`, out)

		out = r.cli(t, []string{"unikraft", "instance", "template", "inspect", templateName})
		assert.Regexp(t, `state:\s+template`, out)
		assert.Regexp(t, `image:\s+nginx`, out)
		assert.Regexp(t, `memory:\s+128`, out)

		r.cli(t, []string{"unikraft", "instance", "template", "edit", templateName, "--set", "tags=env-dev"})

		out = r.cli(t, []string{"unikraft", "instance", "template", "inspect", templateName})
		assert.Regexp(t, `state:\s+template`, out)

		r.cli(t, []string{"unikraft", "instance", "template", "delete", templateName})
	})

	t.Run("create-from-template", func(t *testing.T) {
		r := ir.runner(t, true)
		instName := uniq()

		r.cli(t, []string{
			"unikraft", "instance", "create",
			"--output", "quiet",
			"--set", "name=test-base-" + instName,
			"--set", "metro=" + ir.cfg.MetroName,
			"--set", "image=nginx:latest",
			"--set", "autostart=false",
			"--set", "resources.memory=128",
			"--set", "resources.vcpus=1",
		})

		out := r.cli(t, []string{
			"unikraft", "instance", "template", "create", "test-base-" + instName,
			"--output", "template={{ .name }}",
		})
		templateName := strings.TrimSpace(out)

		r.cli(t, []string{
			"unikraft", "instance", "create",
			"--set", "name=test-from-template-" + instName,
			"--set", "metro=" + ir.cfg.MetroName,
			"--set", "template=" + templateName,
		})

		out = r.cli(t, []string{"unikraft", "instance", "inspect", "test-from-template-" + instName})
		assert.Regexp(t, `state:\s+stopp(?:ing|ed)\b`, out)
		assert.Regexp(t, `memory:\s+128`, out)

		r.cli(t, []string{"unikraft", "instance", "template", "delete", templateName})
	})
}
