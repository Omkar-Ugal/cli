// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package kong

import (
	"reflect"

	"github.com/alecthomas/kong"
)

// Optional is a mapper function that makes a field optional.
func Optional() kong.MapperFunc {
	// NOTE: this works with:
	// --optional
	// --optional=value
	// --optional value
	// --optional --other-flag (does not consume --other-flag)
	// but not with:
	// --optional notvalue
	// because the latter is ambiguous with positional arguments.
	return func(ctx *kong.DecodeContext, target reflect.Value) error {
		switch tok := ctx.Scan.Peek(); tok.Type {
		case kong.FlagValueToken:
			r := kong.NewRegistry().RegisterDefaults()
			return r.ForValue(target).Decode(ctx, target)
		case kong.UntypedToken:
			// Don't consume tokens that look like flags (e.g. --sort, -s).
			if s, ok := tok.Value.(string); ok && len(s) > 0 && s[0] == '-' {
				return nil
			}
			r := kong.NewRegistry().RegisterDefaults()
			return r.ForValue(target).Decode(ctx, target)
		default:
			return nil
		}
	}
}
