// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		r := runner(t, true)

		out := r.Run(t, []string{"unikraft", "config", "get"})
		assert.Regexp(t, `profile:\s+\S+`, out)
		assert.Regexp(t, `token:`, out)

		out = r.Run(t, []string{"unikraft", "config", "get", "-o", "json"})
		assert.Regexp(t, `"token":`, out)
		assert.Regexp(t, `"profile":`, out)

		out = r.Run(t, []string{"unikraft", "config", "get", "-o", "yaml"})
		assert.Regexp(t, `token:`, out)
		assert.Regexp(t, `profile:`, out)
	})
}
