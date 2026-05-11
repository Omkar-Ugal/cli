// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	integ "unikraft.com/cli/internal/integration"
)

// TestHelp runs --help tests for all resource types.
func TestHelp(t *testing.T) {
	unikraftPath := integ.BuildUnikraftBinary(t)
	t.Parallel()

	run := func(name string, fn func(*testing.T, string)) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fn(t, unikraftPath)
		})
	}

	run("general", generalHelpTests)
	run("auth", authHelpTests)
	run("instances", instancesHelpTests)
	run("volumes", volumesHelpTests)
	run("services", servicesHelpTests)
	run("certificates", certificatesHelpTests)
	run("images", imagesHelpTests)
	run("resources", resourceHelpTests)
	run("build", buildHelpTests)
	run("config", configHelpTests)
}

// TestVersion checks that `unikraft version` output contains expected fields.
// Uses regexp instead of golden files because version output contains
// environment-specific values (Go version, OS/arch, build time).
func TestVersion(t *testing.T) {
	t.Parallel()
	unikraftPath := integ.BuildUnikraftBinary(t)
	env := integ.NewTestEnv(t, unikraftPath)
	out := env.CLI(t.Context(), t, []string{"unikraft", "version"})

	assert.Regexp(t, `version:\s+\S+`, out)
	assert.Regexp(t, `commit:\s+\S+`, out)
	assert.Regexp(t, `platform:\s+\S+`, out)
	assert.Regexp(t, `go version:\s+go\d+\.\d+`, out)
	assert.Regexp(t, `docs:\s+https://`, out)
	assert.Regexp(t, `issues:\s+https://`, out)
}

// generalHelpTests checks that top-level help and error output stays stable.
// Deterministic and offline.
func generalHelpTests(t *testing.T, unikraftPath string) {
	r := integ.NewTestEnv(t, unikraftPath)
	integ.Gild(t.Context(), t, r.CLI,
		[]string{"unikraft"},
		[]string{"unikraft", "--help"},
		[]string{"unikraft", "invalid"},
		[]string{"unikraft", "--help", "--bad-flag"},
		[]string{"unikraft", "--help", "bad-arg"},
		[]string{"unikraft", "--log-level=fatal", "invalid"},
	)
}

func authHelpTests(t *testing.T, unikraftPath string) {
	r := integ.NewTestEnv(t, unikraftPath)
	integ.Gild(t.Context(), t, r.CLI,
		[]string{"unikraft", "login", "--help"},
		[]string{"unikraft", "logout", "--help"},
		[]string{"unikraft", "profile", "--help"},
		[]string{"unikraft", "profile", "get", "--help"},
		[]string{"unikraft", "profile", "list", "--help"},
		[]string{"unikraft", "profile", "use", "--help"},
		[]string{"unikraft", "metro", "--help"},
		[]string{"unikraft", "metro", "get", "--help"},
		[]string{"unikraft", "metro", "list", "--help"},
	)
}

func instancesHelpTests(t *testing.T, unikraftPath string) {
	r := integ.NewTestEnv(t, unikraftPath)
	integ.Gild(t.Context(), t, r.CLI,
		[]string{"unikraft", "instance", "--help"},
		[]string{"unikraft", "instance", "get", "--help"},
		[]string{"unikraft", "instance", "list", "--help"},
		[]string{"unikraft", "instance", "wait", "--help"},
		[]string{"unikraft", "instance", "create", "--help"},
		[]string{"unikraft", "instance", "edit", "--help"},
		[]string{"unikraft", "instance", "delete", "--help"},
		[]string{"unikraft", "instance", "template", "--help"},
		[]string{"unikraft", "instance", "template", "get", "--help"},
		[]string{"unikraft", "instance", "template", "list", "--help"},
		[]string{"unikraft", "instance", "template", "create", "--help"},
		[]string{"unikraft", "instance", "template", "edit", "--help"},
		[]string{"unikraft", "instance", "template", "delete", "--help"},
		[]string{"unikraft", "instance", "logs", "--help"},
		[]string{"unikraft", "instance", "start", "--help"},
		[]string{"unikraft", "instance", "stop", "--help"},
		[]string{"unikraft", "instance", "suspend", "--help"},
		[]string{"unikraft", "instance", "restart", "--help"},
	)
}

func volumesHelpTests(t *testing.T, unikraftPath string) {
	r := integ.NewTestEnv(t, unikraftPath)
	integ.Gild(t.Context(), t, r.CLI,
		[]string{"unikraft", "volume", "--help"},
		[]string{"unikraft", "volume", "get", "--help"},
		[]string{"unikraft", "volume", "list", "--help"},
		[]string{"unikraft", "volume", "wait", "--help"},
		[]string{"unikraft", "volume", "create", "--help"},
		[]string{"unikraft", "volume", "clone", "--help"},
		[]string{"unikraft", "volume", "import", "--help"},
		[]string{"unikraft", "volume", "edit", "--help"},
		[]string{"unikraft", "volume", "delete", "--help"},
		[]string{"unikraft", "volume", "template", "--help"},
		[]string{"unikraft", "volume", "template", "get", "--help"},
		[]string{"unikraft", "volume", "template", "list", "--help"},
		[]string{"unikraft", "volume", "template", "create", "--help"},
		[]string{"unikraft", "volume", "template", "edit", "--help"},
		[]string{"unikraft", "volume", "template", "delete", "--help"},
	)
}

func servicesHelpTests(t *testing.T, unikraftPath string) {
	r := integ.NewTestEnv(t, unikraftPath)
	integ.Gild(t.Context(), t, r.CLI,
		[]string{"unikraft", "service", "--help"},
		[]string{"unikraft", "service", "get", "--help"},
		[]string{"unikraft", "service", "list", "--help"},
		[]string{"unikraft", "service", "wait", "--help"},
		[]string{"unikraft", "service", "create", "--help"},
		[]string{"unikraft", "service", "edit", "--help"},
		[]string{"unikraft", "service", "delete", "--help"},
	)
}

func certificatesHelpTests(t *testing.T, unikraftPath string) {
	r := integ.NewTestEnv(t, unikraftPath)
	integ.Gild(t.Context(), t, r.CLI,
		[]string{"unikraft", "certificate", "--help"},
		[]string{"unikraft", "certificate", "get", "--help"},
		[]string{"unikraft", "certificate", "list", "--help"},
		[]string{"unikraft", "certificate", "wait", "--help"},
		[]string{"unikraft", "certificate", "create", "--help"},
		[]string{"unikraft", "certificate", "delete", "--help"},
	)
}

func imagesHelpTests(t *testing.T, unikraftPath string) {
	r := integ.NewTestEnv(t, unikraftPath)
	integ.Gild(t.Context(), t, r.CLI,
		[]string{"unikraft", "image", "--help"},
		[]string{"unikraft", "image", "get", "--help"},
		[]string{"unikraft", "image", "list", "--help"},
		[]string{"unikraft", "image", "copy", "--help"},
	)
}

func resourceHelpTests(t *testing.T, unikraftPath string) {
	r := integ.NewTestEnv(t, unikraftPath)
	integ.Gild(t.Context(), t, r.CLI,
		[]string{"unikraft", "resource", "--help"},
		[]string{"unikraft", "resource", "delete", "--help"},
	)
}

func buildHelpTests(t *testing.T, unikraftPath string) {
	r := integ.NewTestEnv(t, unikraftPath)
	integ.Gild(t.Context(), t, r.CLI,
		[]string{"unikraft", "build", "--help"},
	)
}

func configHelpTests(t *testing.T, unikraftPath string) {
	r := integ.NewTestEnv(t, unikraftPath)
	integ.Gild(t.Context(), t, r.CLI,
		[]string{"unikraft", "config", "--help"},
		[]string{"unikraft", "config", "get", "--help"},
	)
}
