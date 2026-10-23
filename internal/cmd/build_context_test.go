//go:build !js

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"unikraft.com/x/build"
)

func TestResolveNamedBuildContext(t *testing.T) {
	t.Run("local directory", func(t *testing.T) {
		dir := t.TempDir()

		got, err := resolveNamedBuildContext("shared", dir)
		require.NoError(t, err)
		require.IsType(t, build.LocalSource{}, got)

		local := got.(build.LocalSource)

		want, err := filepath.Abs(dir)
		require.NoError(t, err)
		require.Equal(t, want, local.Dir)
	})

	t.Run("docker image reference", func(t *testing.T) {
		const ref = "docker-image://alpine:latest"

		got, err := resolveNamedBuildContext("base", ref)
		require.NoError(t, err)
		require.IsType(t, build.FrontendSource{}, got)

		frontend := got.(build.FrontendSource)
		require.Equal(t, ref, frontend.Ref)
	})

	t.Run("OCI layout reference", func(t *testing.T) {
		const ref = "oci-layout:///tmp/layout"

		got, err := resolveNamedBuildContext("base", ref)
		require.NoError(t, err)
		require.IsType(t, build.FrontendSource{}, got)

		frontend := got.(build.FrontendSource)
		require.Equal(t, ref, frontend.Ref)
	})

	t.Run("missing directory", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "does-not-exist")

		_, err := resolveNamedBuildContext("missing", missing)
		require.Error(t, err)
	})

	t.Run("file instead of directory", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "context.txt")
		require.NoError(t, os.WriteFile(file, []byte("test"), 0600))

		_, err := resolveNamedBuildContext("shared", file)
		require.Error(t, err)
	})

	t.Run("HTTPS Git reference", func(t *testing.T) {
		const ref = "https://github.com/unikraft-cloud/examples.git"

		got, err := resolveNamedBuildContext("shared", ref)
		require.NoError(t, err)
		require.IsType(t, build.FrontendSource{}, got)

		frontend := got.(build.FrontendSource)
		require.Equal(t, ref, frontend.Ref)
	})

	t.Run("HTTP reference", func(t *testing.T) {
		const ref = "http://example.com/context.tar.gz"

		got, err := resolveNamedBuildContext("shared", ref)
		require.NoError(t, err)
		require.IsType(t, build.FrontendSource{}, got)

		frontend := got.(build.FrontendSource)
		require.Equal(t, ref, frontend.Ref)
	})

	t.Run("Git protocol reference", func(t *testing.T) {
		const ref = "git://github.com/unikraft-cloud/examples.git"

		got, err := resolveNamedBuildContext("shared", ref)
		require.NoError(t, err)
		require.IsType(t, build.FrontendSource{}, got)

		frontend := got.(build.FrontendSource)
		require.Equal(t, ref, frontend.Ref)
	})

	t.Run("SSH reference", func(t *testing.T) {
		const ref = "ssh://git@example.com/repo.git"

		got, err := resolveNamedBuildContext("shared", ref)
		require.NoError(t, err)
		require.IsType(t, build.FrontendSource{}, got)

		frontend := got.(build.FrontendSource)
		require.Equal(t, ref, frontend.Ref)
	})

	t.Run("unsupported URL scheme", func(t *testing.T) {
		_, err := resolveNamedBuildContext(
			"remote",
			"ftp://example.com/context.tar.gz",
		)
		require.Error(t, err)
		require.ErrorContains(t, err, "unsupported source scheme")
	})

	t.Run("empty source", func(t *testing.T) {
		_, err := resolveNamedBuildContext("shared", "")
		require.Error(t, err)
		require.ErrorContains(t, err, "empty source")
	})

	t.Run("empty image reference", func(t *testing.T) {
		_, err := resolveNamedBuildContext("base", "docker-image://")
		require.Error(t, err)
		require.ErrorContains(t, err, "empty image reference")
	})

	t.Run("empty OCI layout reference", func(t *testing.T) {
		_, err := resolveNamedBuildContext("base", "oci-layout://")
		require.Error(t, err)
		require.ErrorContains(t, err, "empty OCI layout reference")
	})
}
