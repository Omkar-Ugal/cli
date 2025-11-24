// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"
	jujuerrors "github.com/juju/errors"

	"unikraft.com/cli/internal/cmd"
	"unikraft.com/cli/internal/config"
	"unikraft.com/x/colors"
	"unikraft.com/x/log"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var (
		err error

		args   = os.Args[1:]
		stdin  = os.Stdin
		stdout = os.Stdout
		stderr = os.Stderr
	)

	ctx, err = exec(ctx, args, stdin, stdout, stderr)

	switch {
	case err != nil:
		if config.G(ctx).LogType == log.TextType {
			log.G(ctx).Error().Msg(" ")
			log.G(ctx).Error().Msg(colors.ErrorFg("error:"))
		}

		if log.G(ctx).GetLevel() <= log.DebugLevel {
			if juerr, ok := err.(*jujuerrors.Err); ok {
				for _, cause := range juerr.StackTrace() {
					logErr(ctx, cause)
				}
			}
		} else {
			logErr(ctx, err.Error())
		}

		if config.G(ctx).LogType == log.TextType {
			log.G(ctx).Error().Msg(" ")
		}

		os.Exit(1)
	}
}

func logErr(ctx context.Context, msg string) {
	if config.G(ctx).LogType == log.TextType {
		for _, line := range strings.Split(msg, "\n") {
			log.G(ctx).Error().Msgf("  %s", line)
		}
	} else {
		msg = strings.ReplaceAll(msg, "\n\n", " ")
		msg = strings.ReplaceAll(msg, "\n", " ")
		log.G(ctx).Error().Msg(msg)
	}
}

func getMethod(value reflect.Value, name string) reflect.Value {
	method := value.MethodByName(name)
	if !method.IsValid() {
		if value.CanAddr() {
			method = value.Addr().MethodByName(name)
		}
	}
	return method
}

func exec(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) (context.Context, error) {
	var err error

	cli, opts := cmd.NewRootCmd(ctx, stdin, stdout, stderr)
	cli, err = cli.Parse(args)
	if err != nil {
		return ctx, jujuerrors.Annotate(err, "parsing arguments")
	}

	node := cli.Selected()
	if node == nil {
		if len(cli.Path) == 0 {
			return opts.Context, fmt.Errorf("no command selected")
		}

		selected := cli.Path[0].Node()
		if selected.Type == kong.ApplicationNode {
			method := getMethod(selected.Target, "Run")
			if method.IsValid() {
				node = selected
			}
		}

		if node == nil {
			return opts.Context, fmt.Errorf("no command selected")
		}
	}

	return opts.Context, cli.RunNode(node, &opts.Config)
}
