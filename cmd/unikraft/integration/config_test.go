// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	ir := newIntegrationRunner(t)

	t.Run("get", func(t *testing.T) {
		r := ir.runner(t, true)

		out := r.cli(t, []string{"unikraft", "config", "get"})
		assert.Regexp(t, `profile:\s+\S+`, out)
		assert.Regexp(t, `token:`, out)

		out = r.cli(t, []string{"unikraft", "config", "get", "-o", "json"})
		assert.Regexp(t, `"token":`, out)
		assert.Regexp(t, `"profile":`, out)

		out = r.cli(t, []string{"unikraft", "config", "get", "-o", "yaml"})
		assert.Regexp(t, `token:`, out)
		assert.Regexp(t, `profile:`, out)
	})
}
