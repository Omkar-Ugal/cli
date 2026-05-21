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
	t.Run("inspect", func(t *testing.T) {
		r := runner(t, true)
		out := r.Run(t, []string{"unikraft", "image", "inspect", "nginx:latest"})
		assert.Regexp(t, `ref:\s+nginx`, out)
		assert.Regexp(t, `config:`, out)
		assert.Regexp(t, `kernel:`, out)
		assert.Regexp(t, `kernel.dbg:`, out)
	})

	t.Run("copy-inspect-delete", func(t *testing.T) {
		r := runner(t, true)

		imageTag := uniq()
		imageName := r.Config.Profile.Organization + "/nginx-copy:" + imageTag
		imageFull := fmt.Sprintf("%s/%s", r.Config.Metro.Index().Host, imageName)

		r.Run(t, []string{"unikraft", "image", "copy", "nginx:latest", imageFull})

		out := r.Run(t, []string{"unikraft", "image", "inspect", imageFull})
		assert.Regexp(t, `ref:\s+.*`+imageName, out)
		assert.Regexp(t, `config:`, out)
		assert.Regexp(t, `kernel:`, out)
		assert.Regexp(t, `kernel.dbg:`, out)
		r.Run(t, []string{"unikraft", "image", "delete", imageFull})
	})
}
