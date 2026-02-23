// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package telemetry

import "context"

// contextKey is the context key for storing the command path.
type contextKey struct{}

// WithCommand extracts the command path from the context for telemetry.
func WithCommand(ctx context.Context, cmdPath string) context.Context {
	return context.WithValue(ctx, contextKey{}, cmdPath)
}

// CommandFromContext retrieves the command path from the context for telemetry.
func CommandFromContext(ctx context.Context) (string, bool) {
	cmdPath, ok := ctx.Value(contextKey{}).(string)
	return cmdPath, ok
}
