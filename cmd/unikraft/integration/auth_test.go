// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	ir := newIntegrationRunner(t)

	t.Run("flow", func(t *testing.T) {
		r := ir.runner(t, true)

		out := r.cli(t, []string{"unikraft", "login", "--check"})
		assert.Regexp(t, `authentication token found`, out)

		out = r.cli(t, []string{"unikraft", "profile", "list"})
		assert.Regexp(t, `true`, out)

		out = r.cli(t, []string{"unikraft", "metro", "list"})
		assert.Regexp(t, `https?://`, out)

		out = r.cli(t, []string{"unikraft", "logout"})
		assert.Regexp(t, `logout successful`, out)

		r.cli(t, []string{"unikraft", "profile", "list"}, allowFail())
		r.cli(t, []string{"unikraft", "metro", "list"}, allowFail())
	})
}
