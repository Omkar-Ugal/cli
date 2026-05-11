// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImages(t *testing.T) {
	ir := newIntegrationRunner(t)

	t.Run("inspect", func(t *testing.T) {
		r := ir.runner(t, true)
		out := r.cli(t, []string{"unikraft", "image", "inspect", "nginx:latest"})
		assert.Regexp(t, `nginx`, out)
		assert.Regexp(t, `config:`, out)
		assert.Regexp(t, `kernel:`, out)
	})

	t.Run("copy-inspect-delete", func(t *testing.T) {
		if ir.cfg == nil {
			t.Skip("online test requires config, but no config found")
		}

		imageTag := uniq()
		imageName := ir.cfg.Profile.Organization + "/nginx-copy:" + imageTag
		imageFull := fmt.Sprintf("%s/%s", ir.cfg.Metro.Index().Host, imageName)

		r := ir.runner(t, true)

		r.cli(t, []string{"unikraft", "image", "copy", "nginx:latest", imageFull})

		out := r.cli(t, []string{"unikraft", "image", "inspect", imageFull})
		assert.Regexp(t, `nginx`, out)
		assert.Regexp(t, `config:`, out)
		assert.Regexp(t, `kernel:`, out)
		r.cli(t, []string{"unikraft", "image", "delete", imageFull})
	})
}
