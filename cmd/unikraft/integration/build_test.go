// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/containerd/continuity/fs/fstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	ir := newIntegrationRunner(t)

	// NOTE: only erofs is supported for ROM automounting by the Unikraft
	// kernel currently. CPIO ROMs are not automounted (the kernel hardcodes
	// erofs as the fs type for ROM mounts), so the CPIO variant omits
	// the at= mount option.
	for _, romFormat := range []string{"erofs", "cpio"} {
		t.Run("rom-"+romFormat, func(t *testing.T) {
			var romImagePrefix, baseImagePrefix string
			if ir.cfg != nil {
				romImagePrefix = ir.cfg.Profile.Organization + "/rom-" + romFormat + "-e2e"
				baseImagePrefix = ir.cfg.Profile.Organization + "/busybox-rom-" + romFormat + "-e2e"
			}

			// Only erofs ROMs support kernel automounting via at=.
			// For CPIO, we extract manually from the block device.
			entryCmd := []string{"sh", "-c", "cd /tmp && cpio -id < /dev/ukp_rom_myrom && cat hello.txt"}
			if romFormat == "erofs" {
				entryCmd = []string{"cat", "/rom/hello.txt"}
			}

			r := ir.runner(t, true)
			imageTag := uniq()
			instName := uniq()
			romImage := romImagePrefix + ":" + imageTag
			baseImage := baseImagePrefix + ":" + imageTag
			romFlag := "image=" + romImage + ",name=myrom"
			if romFormat == "erofs" {
				romFlag += ",at=/rom"
			}

			require.NoError(t, fstest.Apply(
				fstest.CreateDir("base", 0o755),
				fstest.CreateFile("base/Dockerfile", []byte(`FROM busybox:latest`), 0o644),
				fstest.CreateFile("base/Kraftfile", fmt.Appendf(nil, `
spec: v0.7
name: busybox-rom-%s-e2e
runtime: base-compat:latest
rootfs:
  format: erofs
  source: ./Dockerfile
cmd: ["%s"]
`, romFormat, strings.Join(entryCmd, `", "`)), 0o644),
				// ROM-only image context: just a directory with a text file.
				fstest.CreateDir("rom", 0o755),
				fstest.CreateDir("rom/myrom", 0o755),
				fstest.CreateFile("rom/myrom/hello.txt", []byte("Hello from ROM!\n"), 0o644),
				fstest.CreateFile("rom/Kraftfile", fmt.Appendf(nil, `
spec: v0.7
name: rom-%s-e2e
roms:
  - source: ./myrom
    format: %s
`, romFormat, romFormat), 0o644),
			).Apply(r.Dir))

			r.cli(t, []string{"unikraft", "build", "base", "--output", baseImage})
			r.cli(t, []string{"unikraft", "build", "rom", "--output", romImage})
			r.cli(t, []string{"unikraft", "run", "--name", "test-" + instName, "--metro", ir.cfg.MetroName, "--output", "quiet", "--image", baseImage, "--rom", romFlag})
			r.cli(t, []string{"unikraft", "instance", "wait", "--until", "state==stopped", "--timeout", "10s", "test-" + instName})

			out := r.cli(t, []string{"unikraft", "instance", "logs", "test-" + instName})
			assert.Regexp(t, `Hello from ROM!`, out)

			r.cli(t, []string{"unikraft", "instance", "delete", "test-" + instName})
		})
	}

	t.Run("rom-dir", func(t *testing.T) {
		var baseImagePrefix string
		if ir.cfg != nil {
			baseImagePrefix = ir.cfg.Profile.Organization + "/busybox-romdir-e2e"
		}

		r := ir.runner(t, true)
		imageTag := uniq()
		instName := uniq()
		baseImage := baseImagePrefix + ":" + imageTag

		require.NoError(t, fstest.Apply(
			fstest.CreateDir("base", 0o755),
			fstest.CreateFile("base/Dockerfile", []byte(`FROM busybox:latest`), 0o644),
			fstest.CreateFile("base/Kraftfile", []byte(`
spec: v0.7
name: busybox-romdir-e2e
runtime: base-compat:latest
rootfs:
  format: erofs
  source: ./Dockerfile
cmd: ["cat", "/rom/hello.txt"]
`), 0o644),
			fstest.CreateDir("romdata", 0o755),
			fstest.CreateFile("romdata/hello.txt", []byte("Hello from ROM!\n"), 0o644),
		).Apply(r.Dir))

		r.cli(t, []string{"unikraft", "build", "base", "--output", baseImage})
		r.cli(t, []string{"unikraft", "run", "--name", "test-" + instName, "--metro", ir.cfg.MetroName, "--output", "quiet", "--image", baseImage, "--rom", "dir=romdata,at=/rom"})
		r.cli(t, []string{"unikraft", "instance", "wait", "--until", "state==stopped", "--timeout", "10s", "test-" + instName})

		out := r.cli(t, []string{"unikraft", "instance", "logs", "test-" + instName})
		assert.Regexp(t, `Hello from ROM!`, out)

		r.cli(t, []string{"unikraft", "instance", "delete", "test-" + instName})
	})

	t.Run("busybox", func(t *testing.T) {
		if ir.cfg == nil {
			t.Skip("busybox tests require online config")
		}
		type variant struct {
			name        string
			imagePrefix string
		}
		variants := []variant{
			{
				name:        "registry",
				imagePrefix: ir.cfg.Profile.Organization + "/busybox-e2e",
			},
			{
				name:        "direct-push",
				imagePrefix: ir.cfg.Metro.Index().Host + "/" + ir.cfg.Profile.Organization + "/busybox-e2e",
			},
		}
		for _, format := range []string{"cpio", "erofs"} {
			t.Run(format, func(t *testing.T) {
				for _, v := range variants {
					t.Run(v.name, func(t *testing.T) {
						r := ir.runner(t, true)
						imageTag := uniq()
						instName := uniq()
						image := v.imagePrefix + ":" + imageTag

						require.NoError(t, fstest.Apply(
							fstest.CreateFile("Dockerfile", []byte(`
FROM busybox:latest
RUN echo "unikraft-e2e" > /etc/unikraft-e2e
COPY <<EOF /entrypoint.sh
#!/bin/sh
echo "== BEGIN /etc/unikraft-e2e =="
cat /etc/unikraft-e2e
echo "== END /etc/unikraft-e2e =="
echo "== BEGIN ls /etc/unikraft-e2e =="
ls /etc/unikraft-e2e
echo "== END ls /etc/unikraft-e2e =="
echo "== BEGIN status =="
echo UNIKRAFT_E2E_OK
echo "== END status =="
EOF
RUN chmod +x /entrypoint.sh
`), 0o644),
							fstest.CreateFile("Kraftfile", fmt.Appendf(nil, `
spec: v0.7
name: busybox-e2e
runtime: base-compat:latest
rootfs:
  format: %s
  source: ./Dockerfile
cmd: ["sh", "/entrypoint.sh"]
`, format), 0o644),
						).Apply(r.Dir))

						r.cli(t, []string{"unikraft", "build", ".", "--output", image})

						out := r.cli(t, []string{"unikraft", "image", "inspect", image})
						assert.Regexp(t, `busybox-e2e`, out)

						r.cli(t, []string{"unikraft", "image", "ls", image, "-okv"})
						r.cli(t, []string{"unikraft", "run", "--name", "test-" + instName, "--metro", ir.cfg.MetroName, "--output", "quiet", "--image", image})
						r.cli(t, []string{"unikraft", "instance", "wait", "--until", "state==stopped", "--timeout", "10s", "test-" + instName})

						out = r.cli(t, []string{"unikraft", "instance", "logs", "test-" + instName})
						assert.Regexp(t, `UNIKRAFT_E2E_OK`, out)

						r.cli(t, []string{"unikraft", "instance", "delete", "test-" + instName})

						r.cli(t, []string{"unikraft", "image", "delete", image})
						r.cli(t, []string{"unikraft", "image", "inspect", image}, expectFail())
						r.cli(t, []string{"unikraft", "image", "ls", image}, expectFail())
					})
				}
			})
		}
	})
}
