// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The KraftKit Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package buildkit

import (
	"context"

	"github.com/moby/buildkit/client"
)

type buildkitContextKey struct{}

func WithBuildkitContext(ctx context.Context, c *client.Client) context.Context {
	return context.WithValue(ctx, buildkitContextKey{}, c)
}

func BuildkitFromContext(ctx context.Context) *client.Client {
	if c, ok := ctx.Value(buildkitContextKey{}).(*client.Client); ok {
		return c
	}
	return nil
}
