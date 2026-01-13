// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"fmt"
	"reflect"
	"slices"
)

func CloneField(field Field) Field {
	newField := field
	newField.Subfields = CloneFields(field.Subfields)
	return newField
}

// CloneFields creates a deep copy of the given slice of Fields.
func CloneFields(fields []Field) []Field {
	if fields == nil {
		return nil
	}
	result := make([]Field, 0, len(fields))
	for _, field := range fields {
		newField := field
		newField.Subfields = CloneFields(field.Subfields)
		result = append(result, newField)
	}
	return result
}

func DedupeFields(fields []Field) []Field {
	seen := make(map[string]int)
	result := make([]Field, 0, len(fields))
	for _, field := range fields {
		if idx, ok := seen[field.Name]; ok {
			result[idx].Subfields = DedupeFields(append(result[idx].Subfields, field.Subfields...))
		} else {
			seen[field.Name] = len(result)
			result = append(result, field)
		}
	}
	return result
}

// MergeFields merges the src slice of Fields into the dest slice of Fields.
func MergeFields(dest []Field, src []Field) []Field {
	return append(slices.Clone(dest), src...)
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
				// this field matches exactly, to remove it
				continue
			}
			destField.Subfields = RemoveFields(destField.Subfields, removeField.Subfields)
			if len(destField.Subfields) == 0 && destField.Value == nil {
				// this field has no remaining children or value, to remove it
				continue
			}
		}
		result = append(result, destField)
	}

	return result
}

// FieldsToMap converts a slice of Fields into a map[string]any suitable for
// marshaling into YAML or JSON.
func FieldsToMap(fields []Field) map[string]any {
	return fieldsToMap(fields)
}

func fieldToValue(field Field) any {
	if field.Elem != nil {
		return fieldsToSlice(field.Subfields)
	}
	if len(field.Subfields) > 0 {
		return fieldsToMap(field.Subfields)
	}
	return field.Value
}

func fieldsToMap(fields []Field) map[string]any {
	result := make(map[string]any, len(fields))
	for _, field := range fields {
		result[field.Name] = fieldToValue(field)
	}
	return result
}

func fieldsToSlice(fields []Field) []any {
	result := make([]any, 0, len(fields))
	for _, field := range fields {
		result = append(result, fieldToValue(field))
	}
	return result
}

// MapToFields converts a map[string]any into a slice of Fields, it loads into
// the field Value.
func MapToFields(fields []Field, m map[string]any) ([]Field, []FieldPath, error) {
	return mapToFields(fields, m)
}

func valueToField(field *Field, val any) (*Field, []FieldPath, error) {
	fieldClone := *field
	field = &fieldClone

	var unknown []FieldPath

	if field.Value != nil {
		field.Value = reflect.New(reflect.TypeOf(field.Value)).Elem().Interface()
		if err := decodeStruct(val, &field.Value); err != nil {
			return nil, nil, fmt.Errorf("failed to decode value %v for field %q: %w", val, field.Name, err)
		}
	}

	if field.Elem != nil {
		var valSlice []any
		if val != nil {
			var ok bool
			valSlice, ok = val.([]any)
			if !ok {
				return nil, nil, fmt.Errorf("expected slice for field %s, got %T", field.Name, val)
			}
		}
		subfields, fieldUnknown, err := slicesToFields(field.Elem, valSlice)
		if err != nil {
			return nil, nil, err
		}
		field.Subfields = subfields
		unknown = fieldUnknown
	} else if len(field.Subfields) > 0 {
		var valMap map[string]any
		if val != nil {
			var ok bool
			valMap, ok = val.(map[string]any)
			if !ok {
				return nil, nil, fmt.Errorf("expected map for field %s, got %T", field.Name, val)
			}
		}
		subfields, fieldUnknown, err := MapToFields(field.Subfields, valMap)
		if err != nil {
			return nil, nil, err
		}
		field.Subfields = subfields
		unknown = fieldUnknown
	}

	for i := range unknown {
		unknown[i] = append(FieldPath{field.Name}, unknown[i]...)
	}
	return field, unknown, nil
}

func mapToFields(fields []Field, m map[string]any) ([]Field, []FieldPath, error) {
	fields = slices.Clone(fields)
	used := map[string]struct{}{}
	var unknown []FieldPath
	for i, field := range fields {
		val, ok := m[field.Name]
		if !ok {
			continue
		}
		field, fieldUnknown, err := valueToField(&field, val)
		if err != nil {
			return nil, nil, err
		}
		fields[i] = *field
		unknown = append(unknown, fieldUnknown...)
		used[field.Name] = struct{}{}
	}
	for key := range m {
		if _, ok := used[key]; !ok {
			unknown = append(unknown, FieldPath{key})
		}
	}
	return fields, unknown, nil
}

func slicesToFields(elem *Field, vals []any) ([]Field, []FieldPath, error) {
	if len(vals) == 0 {
		return nil, nil, nil
	}
	fields := make([]Field, 0, len(vals))
	var unknown []FieldPath
	for i, val := range vals {
		elem := *elem
		elem.Name = fmt.Sprintf("%d", i)
		field, fieldUnknown, err := valueToField(&elem, val)
		if err != nil {
			return nil, nil, err
		}
		fields = append(fields, *field)
		unknown = append(unknown, fieldUnknown...)
	}
	return fields, unknown, nil
}

type filterResult int

const (
	filterExclude filterResult = iota
	filterInclude
	filterRecurse
	filterPrune
)

// filterFields filters the given fields based on the provided predicate
// function f. It recursively filters subfields as well.
func filterFields(fields []Field, f func(Field) filterResult) []Field {
	if fields == nil {
		return nil
	}

	result := make([]Field, 0, len(fields))
	for _, field := range fields {
		ff := f(field)
		if ff == filterExclude {
			continue
		}
		if ff == filterInclude {
			result = append(result, field)
			continue
		}
		field.Subfields = filterFields(field.Subfields, f)
		if ff == filterPrune && len(field.Subfields) == 0 {
			continue
		}
		result = append(result, field)
	}
	return result
}

func filterPatchableFields(fields []Field) []Field {
	return filterFields(fields, func(field Field) filterResult {
		if field.Patch != nil {
			return filterInclude
		}
		return filterPrune
	})
}

func filterCreatableFields(fields []Field) []Field {
	return filterFields(fields, func(field Field) filterResult {
		if field.Create != nil {
			return filterInclude
		}
		return filterPrune
	})
}
