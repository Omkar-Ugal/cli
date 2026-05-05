// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"fmt"
	"regexp"
	"testing"
)

func buildTests(t *testing.T, r *testRunner) {
	t.Run("help", func(t *testing.T) {
		r.run(t, []command{
			{args: []string{unikraftCmd, "build", "--help"}},
		})
	})

	var metroName string
	type variant struct {
		name  string
		image string
	}
	var variants []variant
	if r.cfg != nil {
		metroName = r.cfg.MetroName

		variants = []variant{
			{
				name:  "registry",
				image: fmt.Sprintf("%s/busybox-e2e:$UNIQ_IMAGE", r.cfg.Profile.Organization),
			},
			{
				name:  "direct-push",
				image: fmt.Sprintf("%s/%s/busybox-e2e:$UNIQ_IMAGE", r.cfg.Metro.Index().Host, r.cfg.Profile.Organization),
			},
		}
	}

	t.Run("busybox", func(t *testing.T) {
		if r.cfg == nil {
			t.Skip("busybox tests require online config")
		}
		for _, format := range []string{"cpio", "erofs"} {
			t.Run(format, func(t *testing.T) {
				for _, v := range variants {
					t.Run(v.name, func(t *testing.T) {
						r.
							online().
							withCleaners(buildCleaners).
							withContext(map[string]string{
								"Dockerfile": `
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
`,
								"Kraftfile": fmt.Sprintf(`
spec: v0.7
name: busybox-e2e
runtime: base-compat:latest
rootfs:
  format: %s
  source: ./Dockerfile
cmd: ["sh", "/entrypoint.sh"]
`, format),
							}).
							run(t, []command{
								{args: []string{unikraftCmd, "build", ".", "--output", v.image}},
								{args: []string{unikraftCmd, "image", "inspect", v.image}},
								{args: []string{unikraftCmd, "image", "ls", v.image, "-okv"}},
								{args: []string{unikraftCmd, "run", "--name", "test-$UNIQ_INST", "--metro", metroName, "--output", "quiet", "--image", v.image}},
								{args: []string{unikraftCmd, "instance", "wait", "--until", "state==stopped", "--timeout", "10s", "test-$UNIQ_INST"}},
								{args: []string{unikraftCmd, "instance", "logs", "test-$UNIQ_INST"}},
								{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
								{args: []string{unikraftCmd, "image", "delete", v.image}},
								{args: []string{unikraftCmd, "image", "inspect", v.image}, err: errYes},
								{args: []string{unikraftCmd, "image", "ls", v.image}, err: errYes},
							})
					})
				}
			})
		}
	})
}

var buildCleaners = []cleaner{
	{
		// buildkit versions like "version=v0.25.2" or "version=v0.0.0+unknown" change between environments
		pattern: regexp.MustCompile(`\bversion=v\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?\b`),
		repl:    "version=vX.Y.Z",
	},
	{
		// fractional seconds like ".151719804" in timestamps change between runs
		pattern: regexp.MustCompile(`(\d{2}:\d{2}:\d{2})\.\d+`),
		repl:    "${1}",
	},
	{
		// sizes like "35.55MiB" or "128KiB" change between builds
		pattern: regexp.MustCompile(`\b\d+(\.\d+)?(MiB|KiB|GiB)\b`),
		repl:    "X.XXX${2}",
	},
}
