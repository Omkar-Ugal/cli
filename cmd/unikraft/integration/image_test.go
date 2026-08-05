// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImages(t *testing.T) {
	t.Run("inspect", func(t *testing.T) {
		r := runner(t, true, []string{staging, stable})
		out := r.Run(t, []string{"unikraft", "image", "inspect", "nginx:latest"})
		assert.Regexp(t, `ref:\s+nginx`, out)
		assert.Regexp(t, `config:`, out)
		assert.Regexp(t, `kernel:`, out)
		assert.Regexp(t, `kernel.dbg:`, out)
	})

	t.Run("copy-inspect-delete", func(t *testing.T) {
		r := runner(t, true, []string{staging, stable})

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

	t.Run("list-namespace", func(t *testing.T) {
		r := runner(t, true, []string{staging, stable})
		namespace := r.Config.Profile.Organization
		imageName := namespace + "/nginx-ns:" + uniq()
		imageFull := fmt.Sprintf("%s/%s", r.Config.Metro.Index().Host, imageName)
		r.Run(t, []string{"unikraft", "image", "copy", "nginx:latest", imageFull})

		out := r.Run(t, []string{"unikraft", "image", "list"})
		assert.Contains(t, out, imageName, "unfiltered listing should include the image")

		out = r.Run(t, []string{"unikraft", "image", "list", "--namespace", namespace})
		assert.Contains(t, out, imageName)

		// A different namespace excludes it.
		out = r.Run(t, []string{"unikraft", "image", "list", "--namespace", "official"})
		assert.NotContains(t, out, imageName)

		// The match is exact rather than a prefix, so a namespace that
		// merely extends ours must not select the image either.
		out = r.Run(t, []string{"unikraft", "image", "list", "--namespace", namespace + "x"})
		assert.NotContains(t, out, imageName)

		// The flag is repeatable and unions its values.
		out = r.Run(t, []string{
			"unikraft", "image", "list",
			"--namespace", "official",
			"--namespace", namespace,
		})
		assert.Contains(t, out, imageName)
	})

	t.Run("copy-local-archive", func(t *testing.T) {
		r := runner(t, true, []string{staging, stable})
		archive := filepath.Join(t.TempDir(), "nginx.oci.tar")
		imageTag := uniq()
		imageName := r.Config.Profile.Organization + "/nginx-archive:" + imageTag
		imageFull := fmt.Sprintf("%s/%s", r.Config.Metro.Index().Host, imageName)

		r.Run(t, []string{"unikraft", "image", "copy", "nginx:latest", archive})
		assert.FileExists(t, archive)

		out := r.Run(t, []string{"unikraft", "image", "inspect", archive})
		assert.Regexp(t, `digest:\s+sha256:`, out)
		assert.Regexp(t, `config:`, out)

		r.Run(t, []string{"unikraft", "image", "copy", archive, imageFull})
		out = r.Run(t, []string{"unikraft", "image", "inspect", imageFull})
		assert.Regexp(t, `ref:\s+.*`+imageName, out)

		r.Run(t, []string{"unikraft", "image", "delete", imageFull})
	})
}
