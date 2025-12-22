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
	jujuerrors "github.com/juju/errors"
	"unikraft.com/x/kingkong"
	"unikraft.com/x/log"

	"unikraft.com/cli/internal/cmd/instances"
	"unikraft.com/cli/internal/cmd/login"
	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/version"
)

type UnikraftCLI struct {
	config.Config

	Version version.VersionFlag `group:"flag-global" short:"v" name:"version" help:"Print version information." env:"-"`

	Instances instances.InstancesCmd `cmd:"" help:"View and manage applications."`
	Login     login.LoginCmd         `cmd:"" help:"Login to Unikraft Cloud."`
	Logout    login.LogoutCmd        `cmd:"" help:"Logout from Unikraft Cloud."`
}

func NewRootCmd(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) (*kong.Context, *UnikraftCLI, error) {
	cli := UnikraftCLI{}

	helpOptions := kong.HelpOptions{
		Compact:             true,
		FlagsLast:           true,
		NoExpandSubcommands: true,
	}
	globalFlagGroup := kong.Group{
		Key:   "flag-global",
		Title: kingkong.Underline("Global flags") + ":",
	}

	configFile := filepath.Join(config.ConfigDir(), config.DefaultConfigFilename)

	parser, err := kong.New(&cli,
		kong.Name("unikraft"),
		kong.DefaultEnvars("UNIKRAFT"),
		kong.UsageOnError(),
		kong.Writers(stdout, stderr),
		kong.ConfigureHelp(helpOptions),
		kong.Configuration(kongyaml.Loader, configFile),
		kong.Help(kingkong.HelpPrinter(version.Version)),
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
					Title: kingkong.Underline("Global flags") + ":",
				}
			}

			return nil
		}),
		kong.ExplicitGroups([]kong.Group{
			globalFlagGroup,
			{
				Key:   "flag-local",
				Title: kingkong.Underline("Subcommand flags") + ":",
			},
		}),
	)
	if err != nil {
		return nil, &cli, jujuerrors.Annotate(err, "initializing kong context")
	}

	kctx, err := parser.Parse(args)
	if err != nil {
		return nil, &cli, jujuerrors.Annotate(err, "parsing arguments")
	}

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

	return kctx, &cli, nil
}
