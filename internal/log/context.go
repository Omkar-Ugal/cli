// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

// Package log provides logging facilities.
package log

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
	"unikraft.com/cli/internal/colors"
)

// G is a shorthand for FromContextOrDefault.
// It enables a logging API similar to [containerd/log].
// [containerd/log]: https://pkg.go.dev/github.com/containerd/log
var G = FromContextOrDefault

// contextKey is how we find Loggers in a context.Context.
type contextKey struct{}

// FromContextOrDefault returns a Logger from ctx. If no Logger is found, this
// returns the default Logger.
func FromContextOrDefault(ctx context.Context) *Logger {
	if v, ok := ctx.Value(contextKey{}).(*Logger); ok {
		return v
	}

	return New(os.Stdout, TextType, InfoLevel)
}

// WithLogger returns a new Context, derived from ctx, which carries the
// provided Logger.
func WithLogger(ctx context.Context, v *Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, v)
}

// New returns a slog.Logger backed by a JSON or text handler.
func New(sink io.Writer, typ Type, level Level) *Logger {
	var logger zerolog.Logger

	switch typ {
	case JSONType:
		logger = zerolog.New(sink)
	case TextType:
		fallthrough
	default:
		logger = zerolog.New(zerolog.ConsoleWriter{
			Out: sink,
			FormatLevel: func(i interface{}) string {
				if ll, ok := i.(string); ok {
					level, _ := zerolog.ParseLevel(ll)
					fl, ok := formattedLevels[level]
					if ok {
						return levelColors[level](fmt.Sprintf("%s", fl))
					}
					return ll
				}
				if i == nil {
					return "?"
				}
				return fmt.Sprintf("%s", i)
			},
			FormatFieldName: func(i interface{}) string {
				return colors.InfoFg(fmt.Sprintf("%s=", i))
			},
			FormatMessage: func(i interface{}) string {
				return fmt.Sprintf("%s", i)
			},
			PartsExclude: []string{"time"},
		})
	}

	logger = logger.Level(level).With().Timestamp().Logger()
	return &logger
}

// FormattedLevels are used by ConsoleWriter's consoleDefaultFormatLevel
// for a short level name.
var formattedLevels = map[Level]string{
	TraceLevel: "T",
	DebugLevel: "D",
	InfoLevel:  "i",
	WarnLevel:  "W",
	ErrorLevel: "E",
	FatalLevel: "!",
	PanicLevel: "!",
}

var levelColors = map[Level]func(str ...string) string{
	TraceLevel: lipgloss.NewStyle().Background(colors.Info).Foreground(colors.Info).Render,
	DebugLevel: lipgloss.NewStyle().Background(colors.Success).Foreground(colors.Success).Render,
	InfoLevel:  lipgloss.NewStyle().Background(colors.Primary).Foreground(colors.Primary).Render,
	WarnLevel:  lipgloss.NewStyle().Background(colors.Warning).Foreground(colors.Warning).Render,
	ErrorLevel: lipgloss.NewStyle().Background(colors.Error).Foreground(colors.Error).Render,
	FatalLevel: lipgloss.NewStyle().Background(colors.Error).Foreground(colors.Error).Render,
	PanicLevel: lipgloss.NewStyle().Background(colors.Error).Foreground(colors.Error).Render,
}
