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

	var busybox, metroName string
	if r.cfg != nil {
		metroName = r.cfg.MetroName

		busybox = fmt.Sprintf("%s/busybox-e2e:$UNIQ_IMAGE", r.cfg.Profile.Organization)
		// this is what we'd use to test direct push
		// busybox := fmt.Sprintf("%s/%s/busybox-e2e:$UNIQ_IMAGE", cfg.Metro.Index().Host, cfg.Profile.Organization)
	}

	t.Run("busybox", func(t *testing.T) {
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
				"Kraftfile": `
spec: v0.7
name: busybox-e2e
runtime: base-compat:latest
rootfs: ./Dockerfile
cmd: ["sh", "/entrypoint.sh"]
`,
			}).
			run(t, []command{
				{args: []string{unikraftCmd, "build", ".", "--output", busybox}},
				{args: []string{unikraftCmd, "run", "--name", "test-$UNIQ_INST", "--metro", metroName, "--output", "quiet", "--image", busybox}},
				{args: []string{unikraftCmd, "instance", "wait", "--until", "state==stopped", "--timeout", "10s", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "instance", "logs", "test-$UNIQ_INST"}},
				{args: []string{unikraftCmd, "instance", "delete", "test-$UNIQ_INST"}},
			})
	})
}

var buildCleaners = []cleaner{
	{
		// buildkit versions like "version=v0.25.2" or "version=v0.0.0+unknown" change between environments
		pattern: regexp.MustCompile(`\bversion=v\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?\b`),
		repl:    "version=vX.Y.Z",
	},
}
