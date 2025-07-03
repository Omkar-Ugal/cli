// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"
	"io"
	"path/filepath"
	"runtime"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/log"
	"unikraft.com/cli/internal/version"
)

type UnikraftCLI struct {
	config.Config

	Version version.VersionFlag `group:"flag-global" short:"v" name:"version" help:"Print version information." env:"-"`
}

func NewRootCmd(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) (*kong.Context, *UnikraftCLI) {
	cli := UnikraftCLI{}

	kctx := kong.Parse(&cli,
		kong.Name("unikraft"),
		kong.DefaultEnvars("UNIKRAFT"),
		kong.UsageOnError(),
		kong.Writers(stdout, stderr),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			FlagsLast:           true,
			NoExpandSubcommands: true,
		}),
		kong.Configuration(kongyaml.Loader, filepath.Join(config.ConfigDir(), config.DefaultConfigFilename)),
		kong.Help(helpPrinter),
		kong.WithBeforeReset(func(value *kong.Path) error {
			if value == nil || value.App == nil || value.App.Flags == nil {
				return nil
			}

			for _, f := range value.App.Flags {
				if f.Name != "help" {
					continue
				}

				f.Group = &kong.Group{
					Key:   "flag-global",
					Title: underline("Global flags") + ":",
				}
			}

			return nil
		}),
		kong.ExplicitGroups([]kong.Group{
			{
				Key:   "flag-global",
				Title: underline("Global flags") + ":",
			},
			{
				Key:   "flag-local",
				Title: underline("Subcommand flags") + ":",
			},
		}),
	)

	cli.Context = ctx
	cli.Stdin = stdin
	cli.Stdout = stdout
	cli.Stderr = stderr

	level := log.InfoLevel

	switch cli.Config.LogLevel.String() {
	case "trace":
		level = log.TraceLevel
	case "debug":
		level = log.DebugLevel
	case "info":
		level = log.InfoLevel
	case "warn":
		level = log.WarnLevel
	case "error":
		level = log.ErrorLevel
	case "fatal":
		level = log.FatalLevel
	case "panic":
		level = log.PanicLevel
	default:
		level = log.InfoLevel
	}

	cli.Context = log.WithLogger(cli.Context, log.New(stdout, cli.Config.LogType, level))
	cli.Context = config.WithConfig(cli.Context, &cli.Config)

	log.G(cli.Context).
		Debug().
		Str("version", version.Version).
		Str("arch", runtime.GOARCH).
		Str("plat", runtime.GOOS).
		Str("commit", version.Commit).
		Str("built", version.BuildTime).
		Msg("unikraft CLI")

	return kctx, &cli
}
