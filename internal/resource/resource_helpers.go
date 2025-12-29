// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"fmt"
	"iter"
	"slices"
	"strings"
)

func filterFields(fields []Field, f func(Field, []string) bool, path []string) []Field {
	if fields == nil {
		return nil
	}

	result := make([]Field, 0, len(fields))
	for _, field := range fields {
		path := append(slices.Clone(path), field.Name)
		field.Subfields = filterFields(field.Subfields, f, path)
		if len(field.Subfields) > 0 || f(field, path) {
			result = append(result, field)
		}
	}
	return result
}

func filterPatchableFields(fields []Field) []Field {
	return filterFields(fields, func(field Field, path []string) bool {
		return field.Patch != Patch{}
	}, nil)
}

func filterCreatableFields(fields []Field) []Field {
	return filterFields(fields, func(field Field, path []string) bool {
		return field.Create != Patch{}
	}, nil)
}

func walkFields(fields []Field, parentPath []string, fn func(string, *Field) error) error {
	for idx := range fields {
		field := &fields[idx]
		path := append(append([]string{}, parentPath...), field.Name)

		key := strings.Join(path, ".")
		if err := fn(key, field); err != nil {
			return err
		}

		err := walkFields(field.Subfields, path, fn)
		if err != nil {
			return err
		}
	}
	return nil
}

func IterFields(fields []Field) iter.Seq2[string, *Field] {
	stopErr := fmt.Errorf("stop iteration")
	return func(yield func(string, *Field) bool) {
		walkFields(fields, nil, func(path string, field *Field) error {
			if !yield(path, field) {
				return stopErr
			}
			return nil
		})
	}
}

func CloneFields(fields []Field) []Field {
	result := make([]Field, 0, len(fields))
	for _, field := range fields {
		newField := field
		newField.Subfields = CloneFields(field.Subfields)
		result = append(result, newField)
	}
	return result
}

func LookupField(fields []Field, name string) *Field {
	stopErr := fmt.Errorf("stop iteration")
	var result *Field
	_ = walkFields(fields, nil, func(path string, field *Field) error {
		if path == name {
			result = field
			return stopErr
		}
		return nil
	})
	return result
}

func unpackFields(fields []Field) []Field {
	result := make([]Field, 0, len(fields))
	for _, field := range IterFields(fields) {
		result = append(result, *field)
	}
	return result
}

func fieldValues(fields []Field) map[string]any {
	result := make(map[string]any, len(fields))
	for _, field := range fields {
		if len(field.Subfields) > 0 {
			result[field.Name] = fieldValues(field.Subfields)
		} else {
			result[field.Name] = field.Value
		}
	}
	return result
}
