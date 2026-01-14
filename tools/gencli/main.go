// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"unikraft.com/x/kingkong"
	"unikraft.com/x/log"
)

// CLI is the root command structure.
type CLI struct {
	LogLevel log.Level `group:"flag-global" name:"log-level"  enum:"trace,debug,info,warn,error,fatal" placeholder:"level" default:"info"`
	LogType  log.Type  `group:"flag-global" name:"log-type"  enum:"text,json" placeholder:"type" default:"text"`

	Docs DocsCmd `cmd:"" help:"Generate markdown documentation for the CLI."`
	Man  ManCmd  `cmd:"" help:"Generate man pages for the CLI."`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var cli CLI
	kctx := kong.Parse(&cli,
		kong.Name("gencli"),
		kong.Help(kingkong.HelpPrinter("")),
		kong.Description("Generate documentation and man pages for the Unikraft CLI."),
		kong.UsageOnError(),
	)

	ctx = log.WithLogger(ctx, log.New(os.Stderr, cli.LogType, cli.LogLevel))
	kctx.BindTo(ctx, (*context.Context)(nil))

	if err := kctx.Run(); err != nil {
		kctx.FatalIfErrorf(err)
	}
}
