// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"
	ctrdlog "github.com/containerd/log"
	kongcompletion "github.com/jotaen/kong-completion"
	jujuerrors "github.com/juju/errors"
	"github.com/sirupsen/logrus"
	"unikraft.com/x/kingkong"
	"unikraft.com/x/log"

	"unikraft.com/cli/internal/cmd/login"
	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/logfmt"
	"unikraft.com/cli/internal/resource"
	"unikraft.com/cli/internal/resource/cmd"
	"unikraft.com/cli/internal/version"
	xkong "unikraft.com/cli/internal/x/kong"
	xmaps "unikraft.com/cli/internal/x/maps"
)

type UnikraftCLI struct {
	config.Config

	Version version.VersionFlag `group:"flag-global" name:"version" help:"Print version information." env:"-"`

	Completion kongcompletion.Completion `cmd:"" completion-shell-default:"false" help:"Outputs shell code for initialising tab completions."`

	Login   login.LoginCmd  `cmd:"" help:"Login to Unikraft Cloud."`
	Logout  login.LogoutCmd `cmd:"" help:"Logout from Unikraft Cloud."`
	Profile ProfileCmd      `cmd:"" help:"Manage Unikraft Cloud profiles." aliases:"profile,profiles"`
	Run     RunCmd          `cmd:"" help:"Run an image as an instance."`

	Metros       MetrosCmd       `cmd:"" help:"Manage Unikraft Cloud metros." aliases:"metro,metros"`
	Instances    InstancesCmd    `cmd:"" help:"Manage Unikraft Cloud instances." aliases:"instance,instances,vm,vms"`
	Volumes      VolumesCmd      `cmd:"" help:"Manage Unikraft Cloud volumes." aliases:"volume,volumes,vol,vols"`
	Services     ServicesCmd     `cmd:"" help:"Manage Unikraft Cloud services." aliases:"service,services,svc,svcs"`
	Certificates CertificatesCmd `cmd:"" help:"Manage Unikraft Cloud certificates." aliases:"certificate,certificates,crt,crts,cert,certs"`
	Images       ImagesCmd       `cmd:"" help:"Manage Unikraft Cloud images." aliases:"image,images,img,imgs"`
}

func (cli UnikraftCLI) Examples() []kingkong.Example {
	return []kingkong.Example{
		{
			Description: "Login to Unikraft Cloud",
			Commands: []string{
				"unikraft login",
			},
		},
		{
			Description: "List instances across metros",
			Commands: []string{
				"unikraft instances list",
			},
		},
		{
			Description: "Deploy a new instance from an image",
			Commands: []string{
				"unikraft run --metro=sfo --autostart -p 443:8080/http+tls --scale-to-zero policy=on nginx:latest",
			},
		},
		{
			Description: "Switch to a different profile",
			Commands: []string{
				"unikraft profile use my-other-profile",
			},
		},
	}
}

func NewRootCmd(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) (*kong.Context, *UnikraftCLI, func() error, error) {
	cli := UnikraftCLI{
		Config: config.Config{
			Context: ctx,
			Stdin:   stdin,
			Stdout:  stdout,
			Stderr:  stderr,
		},
	}

	parser, err := NewParser(&cli)
	if err != nil {
		return nil, nil, nil, jujuerrors.Annotate(err, "creating parser")
	}
	parser.Stdout = stdout
	parser.Stderr = stderr

	kongcompletion.Register(
		parser,
		kongcompletion.WithPredictor("resource-key-profile", cmd.PredictResourceKey[Profile](ctx)),
		kongcompletion.WithPredictor("resource-key-metro", cmd.PredictResourceKey[Metro](ctx)),
		kongcompletion.WithPredictor("resource-key-instance", cmd.PredictResourceKey[Instance](ctx)),
		kongcompletion.WithPredictor("resource-key-volume", cmd.PredictResourceKey[Volume](ctx)),
		kongcompletion.WithPredictor("resource-key-service", cmd.PredictResourceKey[ServiceGroup](ctx)),
		kongcompletion.WithPredictor("resource-key-certificate", cmd.PredictResourceKey[Certificate](ctx)),
		kongcompletion.WithPredictor("resource-key-image", cmd.PredictResourceKey[ImageEntry](ctx)),
	)

	kctx, err := parser.Parse(args)

	var parseErr *kong.ParseError
	printHelp := false
	if errors.As(err, &parseErr) {
		// we couldn't parse everything, but still try and manually load these
		// flags, since they're used for help + logging
		for _, flag := range parseErr.Context.Flags() {
			switch flag.Name {
			case "help":
				printHelp = flag.Set
			case "log-type":
				cli.LogType = parseErr.Context.FlagValue(flag).(log.Type)
			case "log-level":
				cli.LogLevel = parseErr.Context.FlagValue(flag).(log.Level)
			}
		}

		// HACK: kong provides UsageOnError, but this shows help for *all* parse
		// errors - we only want to show it only for parent commands.
		// See https://github.com/alecthomas/kong/issues/33
		if strings.HasPrefix(parseErr.Error(), "expected one of") {
			printHelp = true
		}
	}

	if err != nil {
		cli.Context = ctxWithLogger(cli.Context, stderr, cli.LogType, cli.LogLevel)
		cli.Context = config.WithConfig(cli.Context, &cli.Config)
		if printHelp {
			_ = parseErr.Context.PrintUsage(false)
			fmt.Fprintln(os.Stdout)
		}
		return nil, &cli, nil, jujuerrors.Annotate(err, "parsing arguments")
	}

	cli.Context = ctxWithLogger(cli.Context, stderr, cli.LogType, cli.LogLevel)
	cli.Context = config.WithConfig(cli.Context, &cli.Config)
	kctx.BindTo(cli.Context, (*context.Context)(nil))

	sandbox, err := resource.LoadSandboxFromEnv(SandboxedResources...)
	if err != nil {
		return nil, nil, nil, jujuerrors.Annotate(err, "loading sandbox from environment")
	}
	if sandbox != nil {
		sandboxed := xmaps.OrderedKeys(sandbox.Keys)
		slices.Sort(sandboxed)
		log.G(cli.Context).Debug().
			Str("path", sandbox.Path).
			Strs("resources", sandboxed).
			Msg("loaded sandbox from environment")
	}
	kctx.Bind(sandbox)

	cleanup := func() error {
		if err := sandbox.Save(); err != nil {
			return jujuerrors.Annotate(err, "saving sandbox")
		}
		return nil
	}

	log.G(cli.Context).
		Debug().
		Str("version", version.Version).
		Str("arch", runtime.GOARCH).
		Str("plat", runtime.GOOS).
		Str("commit", version.Commit).
		Str("built", version.BuildTime).
		Msg("unikraft CLI")

	return kctx, &cli, cleanup, nil
}

func ctxWithLogger(ctx context.Context, out io.Writer, type_ log.Type, level log.Level) context.Context {
	switch level.String() {
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
	ctx = log.WithLogger(ctx, logfmt.New(out, type_, level))
	ctx = ctrdlog.WithLogger(ctx, logrus.NewEntry(log.ToLogrus(
		log.G(ctx),
		log.WithLogrusLevelCap(logrus.DebugLevel),
	)))
	return ctx
}

func NewParser(cli *UnikraftCLI) (*kong.Kong, error) {
	helpOptions := kong.HelpOptions{
		Compact:             true,
		FlagsLast:           true,
		NoExpandSubcommands: true,
	}
	globalFlagGroup := kong.Group{
		Key:   "flag-global",
		Title: kingkong.Underline("Global flags") + ":",
	}

	// Replace the global logger getter with our own which leverages our own
	// log formatter and configuration.
	log.G = func(ctx context.Context) *log.Logger {
		if v, ok := ctx.Value(log.ContextKey{}).(*log.Logger); ok {
			return v
		}

		return logfmt.New(cli.Stderr, cli.LogType, cli.LogLevel)
	}

	configFile := filepath.Join(config.ConfigDir(), config.DefaultConfigFilename)

	parser, err := kong.New(cli,
		kong.Name("unikraft"),
		kong.DefaultEnvars("UNIKRAFT"),
		kong.UsageOnError(),
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
		kong.NamedMapper("optional", xkong.Optional()),
	)
	return parser, err
}

var SandboxedResources = []resource.Resource{
	Certificate{},
	Instance{},
	ServiceGroup{},
	Volume{},
}

type staticKey string

func (s staticKey) String() string {
	return string(s)
}
