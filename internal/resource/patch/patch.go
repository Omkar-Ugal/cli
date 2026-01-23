// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package patch

import (
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"

	"unikraft.com/cli/internal/resource"
	"unikraft.com/cli/internal/resource/value"
)

type PatchSpec struct {
	Create bool

	Set map[string][]string
	Add map[string][]string
	Del map[string][]string
}

func (spec *PatchSpec) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for key := range spec.Set {
			if !yield(key) {
				return
			}
		}
		for key := range spec.Add {
			if !yield(key) {
				return
			}
		}
		for key := range spec.Del {
			if !yield(key) {
				return
			}
		}
	}
}

// PatchedFields applies the given PatchSpec to the provided fields, returning
// only the modified fields or an error if the patching process encounters
// issues.
func PatchedFields(fields []resource.Field, spec PatchSpec) ([]resource.Field, error) {
	foundFields := make(map[string]struct{})
	unsetFields := make(map[string]struct{})
	setForbiddenFields := make(map[string]struct{})
	addForbiddenFields := make(map[string]struct{})
	delForbiddenFields := make(map[string]struct{})

	fields = resource.CloneFields(fields)
	for key, field := range resource.IterFields(fields) {
		keyStr := key.String()
		foundFields[keyStr] = struct{}{}

		var original *resource.Patch
		var patch **resource.Patch
		if spec.Create {
			original = field.Create
			field.Create = nil
			patch = &field.Create
		} else {
			original = field.Edit
			field.Edit = nil
			patch = &field.Edit
		}
		if original == nil {
			original = &resource.Patch{}
		}

		done := false
		if vs, ok := spec.Set[keyStr]; ok {
			done = true
			if original.Set != nil {
				set, err := value.ParseNew(vs, original.Set)
				if err != nil {
					return nil, fmt.Errorf("failed to unpack set value for %s: %w", keyStr, err)
				}
				*patch = &resource.Patch{Set: set}
			} else {
				setForbiddenFields[keyStr] = struct{}{}
			}
		}
		if vs, ok := spec.Add[keyStr]; ok {
			done = true
			if original.Add != nil {
				add, err := value.ParseNew(vs, original.Add)
				if err != nil {
					return nil, fmt.Errorf("failed to unpack add value for %s: %w", keyStr, err)
				}
				*patch = &resource.Patch{Add: add}
			} else {
				addForbiddenFields[keyStr] = struct{}{}
			}
		}
		if vs, ok := spec.Del[keyStr]; ok {
			done = true
			if original.Del != nil {
				del, err := value.ParseNew(vs, original.Del)
				if err != nil {
					return nil, fmt.Errorf("failed to unpack del value for %s: %w", keyStr, err)
				}
				*patch = &resource.Patch{Del: del}
			} else {
				delForbiddenFields[keyStr] = struct{}{}
			}
		}
		if !done && original.Required {
			unsetFields[keyStr] = struct{}{}
		}
	}

	unknownFields := make([]string, 0)
	for key := range spec.Keys() {
		if _, ok := foundFields[key]; !ok {
			unknownFields = append(unknownFields, key)
		}
	}

	var err error
	if len(unknownFields) > 0 {
		err = errors.Join(err, fmt.Errorf("unknown fields: %v", unknownFields))
	}
	if len(unsetFields) > 0 {
		err = errors.Join(err, fmt.Errorf("required values: %v", slices.Collect(maps.Keys(unsetFields))))
	}
	if len(setForbiddenFields) > 0 {
		err = errors.Join(err, fmt.Errorf("fields not settable: %v", slices.Collect(maps.Keys(setForbiddenFields))))
	}
	if len(addForbiddenFields) > 0 {
		err = errors.Join(err, fmt.Errorf("fields not addable: %v", slices.Collect(maps.Keys(addForbiddenFields))))
	}
	if len(delForbiddenFields) > 0 {
		err = errors.Join(err, fmt.Errorf("fields not deletable: %v", slices.Collect(maps.Keys(delForbiddenFields))))
	}
	if err != nil {
		return nil, err
	}

	if spec.Create {
		return FilterCreatableFields(fields), nil
	}
	return FilterPatchableFields(fields), nil
}
