// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResources(t *testing.T) {
	ir := newIntegrationRunner(t)

	t.Run("volume-flow", func(t *testing.T) {
		r := ir.runner(t, true)
		volName := uniq()

		out := r.cli(t, []string{"unikraft", "resource", "create", "--set", "type=volume", "--set", "name=test-" + volName, "--set", "size=10", "--set", "metro=" + ir.cfg.MetroName})
		assert.Regexp(t, `state:\s+available`, out)

		out = r.cli(t, []string{"unikraft", "resource", "get", "volume:" + ir.cfg.MetroName + "/test-" + volName})
		assert.Regexp(t, `size:\s+10`, out)

		out = r.cli(t, []string{"unikraft", "resource", "list"})
		assert.Regexp(t, `volume`, out)

		r.cli(t, []string{"unikraft", "resource", "edit", "volume:" + ir.cfg.MetroName + "/test-" + volName, "--set", "size=20"})

		out = r.cli(t, []string{"unikraft", "resource", "get", "volume:" + ir.cfg.MetroName + "/test-" + volName})
		assert.Regexp(t, `size:\s+20`, out)

		out = r.cli(t, []string{"unikraft", "volume", "get", "test-" + volName})
		assert.Regexp(t, `size:\s+20`, out)

		r.cli(t, []string{"unikraft", "resource", "delete", "--all", "--force"})
		out = r.cli(t, []string{"unikraft", "volume", "ls"})
		assert.Regexp(t, `METRO\s+NAME`, out)
	})
}
