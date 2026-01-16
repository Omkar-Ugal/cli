// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"
	jujuerrors "github.com/juju/errors"

	"unikraft.com/cli/internal/cmd"
	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/logfmt"
	"unikraft.com/x/colors"
	"unikraft.com/x/log"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var (
		err error

		args  = os.Args[1:]
		stdio = config.Stdio{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
	)

	ctx, err = run(ctx, args, stdio)
	if err == nil {
		// catch context cancellation errors, and make sure we show them, even if
		// the command succeeded
		err = ctx.Err()
	}

	if err != nil && !errors.Is(err, context.Canceled) {
		if logfmt.LogType(ctx) == log.TextType {
			log.G(ctx).Error().Msg(" ")
			log.G(ctx).Error().Msg(colors.ErrorFg("error:"))
		}

		if log.G(ctx).GetLevel() <= log.DebugLevel {
			if juerr, ok := err.(*jujuerrors.Err); ok {
				for _, cause := range juerr.StackTrace() {
					logErr(ctx, cause)
				}
			} else {
				logErr(ctx, err.Error())
			}
		} else {
			logErr(ctx, err.Error())
		}

		if logfmt.LogType(ctx) == log.TextType {
			log.G(ctx).Error().Msg(" ")
		}
	}
	if err != nil {
		os.Exit(1)
	}
}

func logErr(ctx context.Context, msg string) {
	if logfmt.LogType(ctx) == log.TextType {
		for line := range strings.SplitSeq(msg, "\n") {
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

func run(ctx context.Context, args []string, stdio config.Stdio) (context.Context, error) {
	ctx, cli, opts, cleanup, err := cmd.NewRootCmd(ctx, args, stdio)
	if err != nil {
		return ctx, err
	}

	node := cli.Selected()
	if node == nil {
		if len(cli.Path) == 0 {
			return ctx, fmt.Errorf("no command selected")
		}

		selected := cli.Path[0].Node()
		if selected.Type == kong.ApplicationNode {
			method := getMethod(selected.Target, "Run")
			if method.IsValid() {
				node = selected
			}
		}

		if node == nil {
			return ctx, fmt.Errorf("no command selected")
		}
	}

	err = cli.RunNode(node, &opts.Config)
	if cleanup != nil {
		err = errors.Join(err, cleanup())
	}
	return ctx, err
}
