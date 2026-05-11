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

func TestCertificates(t *testing.T) {
	ir := newIntegrationRunner(t)

	t.Run("create", func(t *testing.T) {
		r := ir.runner(t, true)
		certNameA := uniq()
		certNameB := uniq()
		certA := integ.GenerateCert(t)
		certB := integ.GenerateCert(t)

		out := r.cli(t, []string{"unikraft", "certificate", "list"})
		assert.Regexp(t, `METRO\s+NAME`, out)

		out = r.cli(t, []string{"unikraft", "certificate", "create", "--set", "name=test-" + certNameA, "--set", "cn=" + certA.CN, "--set", "chain=" + certA.Chain, "--set", "pkey=" + certA.Key, "--set", "metro=" + ir.cfg.MetroName})
		assert.Regexp(t, `name:\s+test-`, out)
		assert.Regexp(t, `state:\s+valid`, out)

		out = r.cli(t, []string{"unikraft", "certificate", "create", "--set", "name=test-" + certNameB, "--set", "cn=" + certB.CN, "--set", "chain=" + certB.Chain, "--set", "pkey=" + certB.Key, "--set", "metro=" + ir.cfg.MetroName})
		assert.Regexp(t, `name:\s+test-`, out)
		assert.Regexp(t, `state:\s+valid`, out)

		out = r.cli(t, []string{"unikraft", "certificate", "list"})
		assert.Regexp(t, `test-.*valid`, out)

		out = r.cli(t, []string{"unikraft", "certificate", "inspect", "test-" + certNameA, "test-" + certNameB})
		assert.Regexp(t, `state:\s+valid`, out)
		assert.Regexp(t, `common-name:`, out)

		out = r.cli(t, []string{"unikraft", "certificate", "delete", "test-" + certNameA, "test-" + certNameB})
		assert.Regexp(t, `test-`, out)
	})
}
