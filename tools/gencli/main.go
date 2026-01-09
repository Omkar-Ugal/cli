// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"github.com/alecthomas/kong"
	"unikraft.com/x/kingkong"
)

// CLI is the root command structure.
type CLI struct {
	Docs DocsCmd `cmd:"" help:"Generate markdown documentation for the CLI."`
	Man  ManCmd  `cmd:"" help:"Generate man pages for the CLI."`
}

func main() {
	var cli CLI
	kctx := kong.Parse(&cli,
		kong.Name("gencli"),
		kong.Help(kingkong.HelpPrinter("")),
		kong.Description("Generate documentation and man pages for the Unikraft CLI."),
		kong.UsageOnError(),
	)

	if err := kctx.Run(); err != nil {
		kctx.FatalIfErrorf(err)
	}
}
