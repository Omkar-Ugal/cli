// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"fmt"

	"unikraft.com/cli/internal/resource"
)

// FieldSpecAll is the special field spec value that selects all fields,
// regardless of verbosity or emptiness.
const FieldSpecAll = "all"

func SelectFields(fields []resource.Field, header bool, verbosity resource.FieldVerbosity, fieldSpecs []string) ([]resource.Field, error) {
	var base []resource.FieldPath
	var include []resource.FieldPath
	var exclude []resource.FieldPath
	for _, field := range fieldSpecs {
		if len(field) == 0 {
			continue
		}
		if field == FieldSpecAll {
			if base == nil {
				base = []resource.FieldPath{}
			}
			continue
		}
		switch field[0] {
		case '+':
			field = field[1:]
			include = append(include, resource.ParseFieldPath(field))
		case '-':
			field = field[1:]
			exclude = append(exclude, resource.ParseFieldPath(field))
		default:
			base = append(base, resource.ParseFieldPath(field))
		}
	}

	var missing []resource.FieldPath

	result, missing := resource.FilterFieldsByPath(fields, base, !header)
	if base == nil {
		result = resource.FilterFields(result, func(field resource.Field) resource.FilterResult {
			// remove fields that are too verbose
			if field.Verbosity < verbosity {
				return resource.FilterExclude
			}
			// remove empty fields
			if !header && field.IsEmpty() {
				return resource.FilterExclude
			}
			return resource.FilterRecurse
		})
	} else {
		result = resource.FilterFields(result, func(field resource.Field) resource.FilterResult {
			// remove invisible fields
			if field.Verbosity == resource.FieldVerbosityInvisible {
				return resource.FilterExclude
			}
			return resource.FilterRecurse
		})
	}

	if len(include) > 0 {
		included, includeMissing := resource.FilterFieldsByPath(fields, include, !header)
		result = resource.MergeFields(result, included)
		missing = append(missing, includeMissing...)
	}

	if len(exclude) > 0 {
		excluded, excludeMissing := resource.FilterFieldsByPath(fields, exclude, !header)
		result = resource.RemoveFields(result, excluded)
		missing = append(missing, excludeMissing...)
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("unknown fields: %v", missing)
	}

	return result, nil
}
