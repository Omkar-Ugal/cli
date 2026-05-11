// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVolumeTemplates(t *testing.T) {
	ir := newIntegrationRunner(t)

	t.Run("template", func(t *testing.T) {
		r := ir.runner(t, true)
		volName := uniq()

		r.cli(t, []string{
			"unikraft", "volume", "create",
			"--output", "quiet",
			"--set", "name=test-" + volName,
			"--set", "metro=" + ir.cfg.MetroName,
			"--set", "size=10",
		})

		out := r.cli(t, []string{
			"unikraft", "volume", "template", "create", "test-" + volName,
			"--output", "template={{ .name }}",
		})
		templateName := strings.TrimSpace(out)

		out = r.cli(t, []string{"unikraft", "volume", "template", "list"})
		assert.Regexp(t, `NAME`, out)

		out = r.cli(t, []string{"unikraft", "volume", "template", "inspect", templateName})
		assert.Regexp(t, `state:\s+template`, out)
		assert.Regexp(t, `size:\s+10`, out)

		r.cli(t, []string{"unikraft", "volume", "template", "edit", templateName, "--set", "tags=env-dev"})

		out = r.cli(t, []string{"unikraft", "volume", "template", "inspect", templateName})
		assert.Regexp(t, `state:\s+template`, out)

		r.cli(t, []string{"unikraft", "volume", "template", "delete", templateName})
	})
}
