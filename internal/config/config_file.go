// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	UNIKRAFT_CONFIG_DIR = "UNIKRAFT_CONFIG_DIR"
	XDG_CONFIG_HOME     = "XDG_CONFIG_HOME"
	XDG_STATE_HOME      = "XDG_STATE_HOME"
	XDG_DATA_HOME       = "XDG_DATA_HOME"
	APP_DATA            = "AppData"
	LOCAL_APP_DATA      = "LocalAppData"
)

// Config path precedence
// 1. UNIKRAFT_CONFIG_DIR
// 2. XDG_CONFIG_HOME
// 3. AppData (windows only)
// 4. HOME
func ConfigDir() string {
	var path string
	if a := os.Getenv(UNIKRAFT_CONFIG_DIR); a != "" {
		path = a
	} else if b := os.Getenv(XDG_CONFIG_HOME); b != "" {
		path = filepath.Join(b, "unikraft")
	} else if c := os.Getenv(APP_DATA); runtime.GOOS == "windows" && c != "" {
		path = filepath.Join(c, "unikraft")
	} else {
		d, _ := os.UserHomeDir()
		path = filepath.Join(d, ".config", "unikraft")
	}

	return path
}

// State path precedence
// 1. XDG_STATE_HOME
// 2. LocalAppData (windows only)
// 3. HOME
func StateDir() string {
	var path string
	if a := os.Getenv(XDG_STATE_HOME); a != "" {
		path = filepath.Join(a, "unikraft")
	} else if b := os.Getenv(LOCAL_APP_DATA); runtime.GOOS == "windows" && b != "" {
		path = filepath.Join(b, "unikraft")
	} else {
		c, _ := os.UserHomeDir()
		path = filepath.Join(c, ".local", "state", "unikraft")
	}

	return path
}

// Data path precedence
// 1. XDG_DATA_HOME
// 2. LocalAppData (windows only)
// 3. HOME
func DataDir() string {
	var path string
	if a := os.Getenv(XDG_DATA_HOME); a != "" {
		path = filepath.Join(a, "unikraft")
	} else if b := os.Getenv(LOCAL_APP_DATA); runtime.GOOS == "windows" && b != "" {
		path = filepath.Join(b, "unikraft")
	} else {
		c, _ := os.UserHomeDir()
		path = filepath.Join(c, ".local", "share", "unikraft")
	}

	return path
}
