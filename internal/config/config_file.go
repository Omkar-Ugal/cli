// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package config

import (
	"os"
	"path/filepath"
)

const (
	UnikraftConfigDirEnv = "UNIKRAFT_CONFIG_DIR"
)

func ConfigDir() string {
	if path := os.Getenv(UnikraftConfigDirEnv); path != "" {
		return path
	}
	if path, err := os.UserConfigDir(); err == nil {
		return filepath.Join(path, "unikraft")
	}
	return ""
}
