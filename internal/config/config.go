// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package config

import (
	"context"
	"io"

	"unikraft.com/cli/internal/log"
)

// DefaultConfigFilename is the default name of the configuration file used by
// the Unikraft CLI.
var DefaultConfigFilename = "config.yaml"

// Config represents the global configuration for the Unikraft CLI.
type Config struct {
	// Hidden configuration.
	Context context.Context `kong:"-" yaml:"-" json:"-"`
	Stdin   io.Reader       `kong:"-" yaml:"-" json:"-"`
	Stdout  io.Writer       `kong:"-" yaml:"-" json:"-"`
	Stderr  io.Writer       `kong:"-" yaml:"-" json:"-"`

	// Global configuration.
	Config string `group:"flag-global" name:"config" short:"c" help:"Set the configuration file." placeholder:"file" type:"yamlfile" yaml:"-" json:"-"`

	// Logging configuration.
	LogLevel log.Level `group:"flag-global" name:"log-level" help:"Set the logging level." enum:"trace,debug,info,warn,error,fatal" placeholder:"level" default:"info" yaml:"-" json:"-"`
	LogType  log.Type  `group:"flag-global" name:"log-type" help:"Set the log type." enum:"text,json" placeholder:"type" default:"text" yaml:"-" json:"-"`
}
