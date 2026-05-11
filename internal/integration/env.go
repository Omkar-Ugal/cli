// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unicode"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"unikraft.com/cli/internal/resource"
)

var (
	buildOnce       sync.Once
	buildBinaryPath string
	buildBinaryErr  error
)

// BuildUnikraftBinary builds the unikraft CLI binary once and returns its path.
// Subsequent calls return the cached path without rebuilding.
func BuildUnikraftBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		binaryDir, err := os.MkdirTemp("", "unikraft-cli-test-*")
		if err != nil {
			buildBinaryErr = fmt.Errorf("create temp dir: %w", err)
			return
		}
		binaryName := "unikraft"
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}
		binaryPath := filepath.Join(binaryDir, binaryName)
		cmd := exec.Command("go", "build", "-buildvcs=false", "-o", binaryPath, "unikraft.com/cli/cmd/unikraft")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			buildBinaryErr = fmt.Errorf("go build failed\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
			return
		}
		buildBinaryPath = binaryPath
	})
	require.NoError(t, buildBinaryErr)
	return buildBinaryPath
}

// TestEnv holds per-test environment for running CLI commands deterministically.
type TestEnv struct {
	UnikraftPath string
	ConfigPath   string
	SandboxPath  string
	Dir          string
}

// NewTestEnv creates a new isolated test environment.
func NewTestEnv(t *testing.T, unikraftPath string) *TestEnv {
	t.Helper()
	dir := t.TempDir()
	return &TestEnv{
		UnikraftPath: unikraftPath,
		ConfigPath:   filepath.Join(dir, "config.yaml"),
		SandboxPath:  filepath.Join(dir, "sandbox.json"),
		Dir:          dir,
	}
}

// execCmd is the shared command execution implementation.
func (env *TestEnv) execCmd(ctx context.Context, t *testing.T, args []string) (string, error) {
	t.Helper()

	t.Logf("executing: %s", strings.Join(args, " "))

	var c *exec.Cmd
	if args[0] == "unikraft" {
		c = exec.CommandContext(ctx, env.UnikraftPath, args[1:]...)
	} else {
		c = exec.CommandContext(ctx, args[0], args[1:]...)
	}

	var output bytes.Buffer
	c.Stdout = &output
	c.Stderr = &output
	c.Dir = env.Dir
	c.Env = os.Environ()
	c.Env = slices.DeleteFunc(c.Env, func(s string) bool {
		return strings.HasPrefix(s, "UNIKRAFT_")
	})
	c.Env = append(c.Env, "NO_COLOR=1")
	c.Env = append(c.Env, "UNIKRAFT_CONFIG="+env.ConfigPath)
	c.Env = append(c.Env, "BUILDKIT_PROGRESS=quiet")
	c.Env = append(c.Env, resource.UnikraftSandboxEnv+"="+env.SandboxPath)

	err := c.Run()
	return ansi.Strip(output.String()), err
}

// RunCmd executes a command and returns the combined output and any error.
func (env *TestEnv) RunCmd(ctx context.Context, t *testing.T, args []string) (string, error) {
	return env.execCmd(ctx, t, args)
}

// CLI runs a CLI command and returns formatted output for golden comparison.
func (env *TestEnv) CLI(ctx context.Context, t *testing.T, args []string) string {
	t.Helper()

	out, err := env.execCmd(ctx, t, args)

	var exitCode int
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		require.NoError(t, err, "command %q failed: %s", strings.Join(args, " "), out)
	}

	var result strings.Builder
	result.WriteString("$ " + strings.Join(FormatArgs(args), " ") + "\n")

	normalized := NormalizeOutput(out)
	if normalized != "" {
		result.WriteString("\n" + normalized + "\n")
	}

	if exitCode != 0 {
		result.WriteString("\nexit code: " + strconv.Itoa(exitCode) + "\n")
	}

	return result.String()
}

// FormatArgs formats command arguments for display, quoting where necessary.
func FormatArgs(args []string) []string {
	formatted := make([]string, 0, len(args))
	for _, arg := range args {
		formatted = append(formatted, QuoteArg(arg))
	}
	return formatted
}

// QuoteArg quotes a single argument if it contains characters that require quoting.
func QuoteArg(arg string) string {
	if arg == "" {
		return "''"
	}
	if strings.ContainsAny(arg, " \t\n{}()") {
		return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
	}
	return arg
}

// NormalizeOutput strips ANSI codes and normalizes line endings.
func NormalizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = ansi.Strip(s)
	s = strings.TrimRightFunc(s, unicode.IsSpace)
	return s
}
