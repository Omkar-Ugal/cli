// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").

package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResources(t *testing.T) {
	t.Run("volume-flow", func(t *testing.T) {
		r := runner(t, true)
		volName := uniq()

		out := r.Run(t, []string{"unikraft", "resource", "create", "--set", "type=volume", "--set", "name=test-" + volName, "--set", "size=10", "--set", "metro=" + r.Config.MetroName})
		assert.Regexp(t, `state:\s+available`, out)

		out = r.Run(t, []string{"unikraft", "resource", "get", "volume:" + r.Config.MetroName + "/test-" + volName})
		assert.Regexp(t, `size:\s+10`, out)

		out = r.Run(t, []string{"unikraft", "resource", "list"})
		assert.Regexp(t, `volume`, out)

		r.Run(t, []string{"unikraft", "resource", "edit", "volume:" + r.Config.MetroName + "/test-" + volName, "--set", "size=20"})

		out = r.Run(t, []string{"unikraft", "resource", "get", "volume:" + r.Config.MetroName + "/test-" + volName})
		assert.Regexp(t, `size:\s+20`, out)

		out = r.Run(t, []string{"unikraft", "volume", "get", "test-" + volName})
		assert.Regexp(t, `size:\s+20`, out)

		r.Run(t, []string{"unikraft", "resource", "delete", "--all", "--force"})
		out = r.Run(t, []string{"unikraft", "volume", "list", "--output", "quiet"})
		assert.Empty(t, strings.TrimSpace(out))
	})
}
