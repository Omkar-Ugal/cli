// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package logfmt

import (
	"context"

	"unikraft.com/x/log"
)

type logTypeContextKey struct{}

func WithLogType(ctx context.Context, typ log.Type) context.Context {
	return context.WithValue(ctx, logTypeContextKey{}, typ)
}

func LogType(ctx any) log.Type {
	if v, ok := ctx.(context.Context).Value(logTypeContextKey{}).(log.Type); ok {
		return v
	}
	return log.TextType // default
}
