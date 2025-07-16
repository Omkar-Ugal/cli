// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package config

import (
	"context"
)

// G is a shorthand for FromContextOrDefault.
// It enables a logging API similar to [containerd/log].
// [containerd/log]: https://pkg.go.dev/github.com/containerd/log
var G = FromContextOrDefault

// contextKey is how we find the config in a context.Context.
type contextKey struct{}

// FromContextOrDefault returns a config from ctx. If no config is found, this
// returns the default config.
func FromContextOrDefault(ctx context.Context) *Config {
	if v, ok := ctx.Value(contextKey{}).(*Config); ok {
		return v
	}

	return &Config{}
}

// WithConfig returns a new Context, derived from ctx, which carries the
// provided global configuration.
func WithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, contextKey{}, cfg)
}
