// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

// CloneFields creates a deep copy of the given slice of Fields.
func CloneFields(fields []Field) []Field {
	result := make([]Field, 0, len(fields))
	for _, field := range fields {
		newField := field
		newField.Subfields = CloneFields(field.Subfields)
		result = append(result, newField)
	}
	return result
}

// MergeFields merges the src slice of Fields into the dest slice of Fields.
// If a field with the same Name exists in both slices, their Subfields are
// merged recursively.
func MergeFields(dest []Field, src []Field) []Field {
	destMap := make(map[string]*Field)
	for i := range dest {
		field := &dest[i]
		destMap[field.Name] = field
	}

	for _, srcField := range src {
		if destField, ok := destMap[srcField.Name]; ok {
			destField.Subfields = MergeFields(destField.Subfields, srcField.Subfields)
		} else {
			dest = append(dest, srcField)
		}
	}

	return dest
}

// RemoveFields removes fields from the dest slice of Fields based on the
// remove slice of Fields. If a field with the same Name exists in both slices,
// it is removed from dest.
func RemoveFields(dest []Field, remove []Field) []Field {
	removeMap := make(map[string]*Field)
	for _, field := range remove {
		removeMap[field.Name] = &field
	}

	result := make([]Field, 0, len(dest))
	for _, destField := range dest {
		if removeField, ok := removeMap[destField.Name]; ok {
			if len(removeField.Subfields) == 0 {
				continue
			}
			destField.Subfields = RemoveFields(destField.Subfields, removeField.Subfields)
			if len(destField.Subfields) == 0 {
				continue
			}
		}
		result = append(result, destField)
	}

	return result
}

// fieldsToMap converts a slice of Fields into a map[string]any suitable for
// marshaling into YAML or JSON.
func fieldsToMap(fields []Field) map[string]any {
	result := make(map[string]any, len(fields))
	for _, field := range fields {
		if len(field.Subfields) > 0 {
			result[field.Name] = fieldsToMap(field.Subfields)
		} else {
			result[field.Name] = field.Value
		}
	}
	return result
}

// filterFields filters the given fields based on the provided predicate
// function f. It recursively filters subfields as well.
func filterFields(fields []Field, f func(Field) bool) []Field {
	if fields == nil {
		return nil
	}

	result := make([]Field, 0, len(fields))
	for _, field := range fields {
		field.Subfields = filterFields(field.Subfields, f)
		if len(field.Subfields) > 0 || f(field) {
			result = append(result, field)
		}
	}
	return result
}

func filterPatchableFields(fields []Field) []Field {
	return filterFields(fields, func(field Field) bool {
		return field.Patch != nil
	})
}

func filterCreatableFields(fields []Field) []Field {
	return filterFields(fields, func(field Field) bool {
		return field.Create != nil
	})
}
