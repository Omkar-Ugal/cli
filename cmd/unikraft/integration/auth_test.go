// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"

	integ "unikraft.com/cli/internal/integration"
)

func TestAuth(t *testing.T) {
	t.Run("flow", func(t *testing.T) {
		r := runner(t, true)

		out := r.Run(t, []string{"unikraft", "login", "--check"})
		assert.Regexp(t, `authentication token found`, out)

		out = r.Run(t, []string{"unikraft", "profile", "list"})
		assert.Regexp(t, `true`, out)

		out = r.Run(t, []string{"unikraft", "metro", "list"})
		assert.Regexp(t, `https?://`, out)

		out = r.Run(t, []string{"unikraft", "logout"})
		assert.Regexp(t, `logout successful`, out)

		r.Run(t, []string{"unikraft", "profile", "list"}, integ.AllowFail())
		r.Run(t, []string{"unikraft", "metro", "list"}, integ.AllowFail())
	})
}
