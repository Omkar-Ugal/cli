// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"cmp"
	"encoding"
	"fmt"
	"reflect"

	"github.com/alecthomas/kong"

	"unikraft.com/cli/internal/resource/value"
)

// ApplyShortcutFlags scans Kong flags for those tagged with
// `shortcut:"<field-path>"` or `shortcut-file:"<field-path>"` and appends
// their values to args as --set or --set-file entries.
//
// Only flags that were explicitly set by the user (flag.Set == true) are
// processed. This allows users to set zero values like --replicas=0 or
// --name= without them being skipped. Slice fields produce one entry per
// element.
func ApplyShortcutFlags(args *SetArgs, flags []*kong.Flag) error {
	for _, flag := range flags {
		if flag == nil || flag.Value == nil || flag.Tag == nil {
			continue
		}

		key := flag.Tag.Get("shortcut")
		fileKey := flag.Tag.Get("shortcut-file")
		if key == "" && fileKey == "" {
			continue
		}
		if key == "-" || fileKey == "-" {
			continue
		}

		// Only process flags that were explicitly set by the user.
		if !flag.Set {
			continue
		}

		fv := flag.Target
		// Dereference pointers to get the underlying value.
		for fv.Kind() == reflect.Ptr {
			fv = fv.Elem()
		}

		// Slice fields produce one --set entry per element, unless the
		// type implements TextUnmarshaler (meaning the slice is a single
		// logical value, like InstanceArgs).
		isTextUnmarshaler := false
		if fv.CanAddr() {
			_, isTextUnmarshaler = fv.Addr().Interface().(encoding.TextUnmarshaler)
		}
		if fv.Kind() == reflect.Slice && !isTextUnmarshaler {
			for j := range fv.Len() {
				elem := fv.Index(j)
				// If the element is addressable, take its address to allow
				// pointer-receiver MarshalText methods to be called.
				var elemVal any
				if elem.CanAddr() {
					elemVal = elem.Addr().Interface()
				} else {
					elemVal = elem.Interface()
				}
				s, err := value.Format(elemVal)
				if err != nil {
					return fmt.Errorf("shortcut %q[%d]: %w", cmp.Or(key, fileKey), j, err)
				}
				if fileKey != "" {
					args.SetFile = append(args.SetFile, map[string]string{fileKey: s})
					continue
				}
				args.Set = append(args.Set, map[string]string{key: s})
			}
			continue
		}

		s, err := value.Format(fv.Interface())
		if err != nil {
			return fmt.Errorf("shortcut %q: %w", cmp.Or(key, fileKey), err)
		}
		if fileKey != "" {
			args.SetFile = append(args.SetFile, map[string]string{fileKey: s})
			continue
		}
		args.Set = append(args.Set, map[string]string{key: s})
	}
	return nil
}
