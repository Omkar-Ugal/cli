// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package patch

import (
	"fmt"
	"reflect"

	"unikraft.com/cli/internal/resource"
)

// collectValue collects a value of the given type from the resource.Field.
//
// In the simplest case, this is just returning field.Value, but for nested
// structures, like lists of fields, then we need to assemble a value from
// the subfields. e.g. [resource.Field{Value: 1}, resource.Field{Value: 2}] -> []int{1, 2}
func collectValue(field resource.Field, into reflect.Type) (any, error) {
	v, err := collectValueRaw(field)
	if err != nil {
		return nil, err
	}
	if v != nil {
		valueType := reflect.TypeOf(v)
		if valueType.AssignableTo(into) {
			return v, nil
		}
		if valueType.ConvertibleTo(into) {
			return reflect.ValueOf(v).Convert(into).Interface(), nil
		}
	}
	target := reflect.New(into).Elem().Interface()

	if err := resource.DecodeStruct(v, &target); err != nil {
		return nil, fmt.Errorf("failed to decode value into type %s: %w", into.String(), err)
	}
	return target, nil
}

func collectValueRaw(field resource.Field) (any, error) {
	if field.Value != nil {
		return field.Value, nil
	}
	// Callback fields without resolved values - return nil gracefully.
	// These fields are typically not editable anyway.
	if field.ValueCallback != nil {
		return nil, nil
	}
	if field.Elem != nil {
		sl := make([]any, 0, len(field.Subfields))
		for _, subfield := range field.Subfields {
			elemValue, err := collectValueRaw(subfield)
			if err != nil {
				return nil, err
			}
			sl = append(sl, elemValue)
		}
		return sl, nil
	}
	if len(field.Subfields) > 0 {
		m := make(map[string]any, len(field.Subfields))
		for _, subfield := range field.Subfields {
			subfieldValue, err := collectValueRaw(subfield)
			if err != nil {
				return nil, err
			}
			m[subfield.Name] = subfieldValue
		}
		return m, nil
	}
	return nil, fmt.Errorf("field has no value")
}

// storeValue stores a value into the given resource.Field, as the reverse of collectValue.
//
// In the simplest case, this is just setting field.Value, but for nested structures,
// like lists of fields, then we need to decompose the value into the subfields.
// e.g. []int{1, 2} -> [resource.Field{Value: 1}, resource.Field{Value: 2}].
func storeValue(field *resource.Field, base reflect.Value) error {
	value := base
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		value = value.Elem()
	}

	if field.Value != nil {
		if base.Type().AssignableTo(reflect.TypeOf(field.Value)) {
			field.Value = base.Interface()
		} else if base.Type().ConvertibleTo(reflect.TypeOf(field.Value)) {
			field.Value = base.Convert(reflect.TypeOf(field.Value)).Interface()
		} else {
			return fmt.Errorf("cannot assign value of type %s to field %s of type %T", value.Type().String(), field.Name, field.Value)
		}
	}

	if field.Elem != nil {
		if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
			return fmt.Errorf("expected slice for field %s, got %s", field.Name, value.Kind().String())
		}
		if field.Subfields == nil {
			field.Subfields = make([]resource.Field, value.Len())
			for i := range field.Subfields {
				field.Subfields[i] = resource.CloneField(*field.Elem)
				field.Subfields[i].Name = fmt.Sprintf("%d", i)
			}
		}
		if value.Len() != len(field.Subfields) {
			return fmt.Errorf("expected %d elements for field %s, got %d", len(field.Subfields), field.Name, value.Len())
		}
		for i := range value.Len() {
			err := storeValue(&field.Subfields[i], value.Index(i))
			if err != nil {
				return err
			}
		}
		return nil
	}
	if len(field.Subfields) > 0 {
		if value.Kind() != reflect.Struct {
			// NOTE: theoretically possible to support more types here
			// but in reality, we don't seem to have Patches for other types
			return fmt.Errorf("expected struct for field %s, got %s", field.Name, value.Kind().String())
		}
		for i := range value.NumField() {
			parsedField, err := resource.ParseField(value.Type().Field(i))
			if err != nil {
				return err
			}
			if parsedField == nil {
				continue
			}

			var subfield *resource.Field
			for j := range field.Subfields {
				if field.Subfields[j].Name == parsedField.Name {
					subfield = &field.Subfields[j]
					break
				}
			}
			if subfield == nil {
				return fmt.Errorf("no subfield named %s in field %s", parsedField.Name, field.Name)
			}
			err = storeValue(subfield, value.Field(i))
			if err != nil {
				return err
			}
		}
		return nil
	}

	if field.Value != nil {
		return nil
	}
	return fmt.Errorf("field has no value")
}
