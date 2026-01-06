// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package config

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	jujuerrors "github.com/juju/errors"
	"gopkg.in/yaml.v3"
	"unikraft.com/x/log"
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
	Config    string `group:"flag-global" name:"config" short:"c" help:"Set the configuration file." placeholder:"file" type:"yamlfile" yaml:"-" json:"-"`
	Telemetry bool   `group:"flag-global" name:"telemetry" help:"Enable or disable telemetry." default:"true" negatable:"" yaml:"telemetry" json:"telemetry"`
	Emojis    bool   `group:"flag-global" name:"emojis" help:"Enable or disable emojis in the CLI output." default:"true" negatable:"" yaml:"emojis" json:"emojis"`

	// Logging configuration.
	LogLevel log.Level `group:"flag-global" name:"log-level" help:"Set the logging level." enum:"trace,debug,info,warn,error,fatal" placeholder:"level" default:"info" yaml:"-" json:"-"`
	LogType  log.Type  `group:"flag-global" name:"log-type" help:"Set the log type." enum:"text,json" placeholder:"type" default:"text" yaml:"-" json:"-"`

	// Profile configuration.
	Profile  string             `group:"flag-global" name:"profile" help:"Set the current profile." placeholder:"name" default:"default"`
	Profiles map[string]Profile `hidden:"" help:"List of profiles." json:"profiles"`
}

// Save the current configuration to the configuration file.
func (c *Config) Save() error {
	return c.SaveTo(ConfigDir())
}

func (c *Config) SaveTo(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return jujuerrors.Annotate(err, "creating config directory")
	}

	// Open the existing configuration file or create a new one.
	configFile := filepath.Join(dir, DefaultConfigFilename)
	f, err := os.OpenFile(configFile, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return jujuerrors.Annotate(err, "opening config file")
	}

	defer f.Close()

	headData, err := yaml.Marshal(c)
	if err != nil {
		return jujuerrors.Annotate(err, "marshalling latest config")
	}

	var headNode yaml.Node
	if err := yaml.Unmarshal(headData, &headNode); err != nil {
		return jujuerrors.Annotate(err, "parsing latest config")
	}
	// Skip document root.
	headNode = *headNode.Content[0]

	// Write the merged configuration back to the file.
	if err := f.Truncate(0); err != nil {
		return jujuerrors.Annotate(err, "truncating config file")
	}
	if _, err := f.Seek(0, 0); err != nil {
		return jujuerrors.Annotate(err, "seeking start of the config file")
	}

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)

	return encoder.Encode(headNode)
}

func LoadFrom(dir string) (*Config, error) {
	configFile := filepath.Join(dir, DefaultConfigFilename)
	f, err := os.Open(configFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, jujuerrors.Annotate(err, "opening config file")
	}
	defer f.Close()

	c := Config{}
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&c); err != nil {
		return nil, jujuerrors.Annotate(err, "decoding config file")
	}

	return &c, nil
}
