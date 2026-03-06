// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package binorigin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	jujuerrors "github.com/juju/errors"
)

// BinaryOrigin represents how the CLI binary was installed.
type BinaryOrigin int

const (
	// OriginUnknown indicates the installation method could not be determined.
	OriginUnknown BinaryOrigin = iota
	// OriginInstallScript indicates installation via install.sh.
	OriginInstallScript
	// OriginAPT indicates installation via APT (Debian/Ubuntu).
	OriginAPT
	// OriginRPM indicates installation via RPM (Fedora/RHEL/Rocky/Alma).
	OriginRPM
	// OriginAPK indicates installation via APK (Alpine).
	OriginAPK
	// OriginBrew indicates installation via Homebrew (macOS).
	OriginBrew
)

// String returns the human-readable name of the origin.
func (o BinaryOrigin) String() string {
	switch o {
	case OriginAPT:
		return "apt"
	case OriginRPM:
		return "rpm"
	case OriginAPK:
		return "apk"
	case OriginBrew:
		return "brew"
	case OriginInstallScript:
		return "install.sh"
	default:
		return "unknown"
	}
}

// PackageManagerCommand returns the command to use for upgrading via the package manager.
func (o BinaryOrigin) PackageManagerCommand() string {
	switch o {
	case OriginAPT:
		return "sudo apt update && sudo apt upgrade unikraft-cli"
	case OriginRPM:
		return "sudo dnf upgrade unikraft-cli"
	case OriginAPK:
		return "sudo apk update && sudo apk upgrade unikraft-cli"
	case OriginBrew:
		return "brew upgrade unikraft"
	default:
		return ""
	}
}

// DetectBinaryOrigin determines how the CLI binary was installed.
func DetectBinaryOrigin(ctx context.Context) (BinaryOrigin, error) {
	execPath, err := os.Executable()
	if err != nil {
		return OriginUnknown, jujuerrors.Annotate(err, "getting executable path")
	}

	// Resolve symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return OriginUnknown, jujuerrors.Annotate(err, "resolving symlinks")
	}

	// Check for Homebrew first (macOS or Linux)
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if origin := isHomebrew(ctx, realPath); origin == OriginBrew {
			return OriginBrew, nil
		}
	}

	// Check for Linux package managers
	if runtime.GOOS == "linux" {
		// Check APT (Debian/Ubuntu)
		if origin := isAPT(ctx, realPath); origin == OriginAPT {
			return OriginAPT, nil
		}

		// Check RPM (Fedora/RHEL/Rocky/Alma)
		if origin := isRPM(ctx, realPath); origin == OriginRPM {
			return OriginRPM, nil
		}

		// Check APK (Alpine)
		if origin := isAPK(ctx, realPath); origin == OriginAPK {
			return OriginAPK, nil
		}
	}

	// If no package manager detected, assume install.sh
	return OriginInstallScript, nil
}

// isHomebrew checks if the binary was installed via Homebrew.
func isHomebrew(ctx context.Context, binaryPath string) BinaryOrigin {
	// Check if path contains Homebrew Cellar
	if strings.Contains(binaryPath, "/Cellar/") || strings.Contains(binaryPath, "/homebrew/") {
		return OriginBrew
	}

	// Try `brew list --formula unikraft` to see if Homebrew manages it
	cmd := exec.CommandContext(ctx, "brew", "list", "--formula", "unikraft")
	if err := cmd.Run(); err == nil {
		return OriginBrew
	}

	return OriginUnknown
}

// isAPT checks if the binary was installed via APT.
func isAPT(ctx context.Context, binaryPath string) BinaryOrigin {
	// Check if dpkg is available and knows about the file
	cmd := exec.CommandContext(ctx, "dpkg", "-S", binaryPath)
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), "unikraft") {
		return OriginAPT
	}

	return OriginUnknown
}

// isRPM checks if the binary was installed via RPM.
func isRPM(ctx context.Context, binaryPath string) BinaryOrigin {
	// Check if rpm is available and knows about the file
	cmd := exec.CommandContext(ctx, "rpm", "-qf", binaryPath)
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), "unikraft") {
		return OriginRPM
	}

	return OriginUnknown
}

// isAPK checks if the binary was installed via APK (Alpine).
func isAPK(ctx context.Context, binaryPath string) BinaryOrigin {
	// Check if apk is available and knows about the file
	cmd := exec.CommandContext(ctx, "apk", "info", "--who-owns", binaryPath)
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), "unikraft") {
		return OriginAPK
	}

	return OriginUnknown
}
