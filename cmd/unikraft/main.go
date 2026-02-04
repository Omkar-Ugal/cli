// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"context"
	"errors"
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

	ctx, err = run(ctx, args, stdin, stdout, stderr)
	if err == nil {
		// catch context cancellation errors, and make sure we show them, even if
		// the command succeeded
		err = ctx.Err()
	}

	if err != nil && !errors.Is(err, context.Canceled) {
		if config.G(ctx).LogType == log.TextType || config.G(ctx).LogType == "" {
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

		if config.G(ctx).LogType == log.TextType || config.G(ctx).LogType == "" {
			log.G(ctx).Error().Msg(" ")
		}
	}
	if err != nil {
		os.Exit(1)
	}
}

func logErr(ctx context.Context, msg string) {
	if config.G(ctx).LogType == log.TextType || config.G(ctx).LogType == "" {
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

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) (context.Context, error) {
	cli, opts, cleanup, err := cmd.NewRootCmd(ctx, args, stdin, stdout, stderr)
	if err != nil {
		if opts != nil && opts.Context != nil {
			return opts.Context, err
		}
		return ctx, err
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

	err = cli.RunNode(node, &opts.Config)
	if cleanup != nil {
		err = errors.Join(err, cleanup())
	}
	return opts.Context, err
}
