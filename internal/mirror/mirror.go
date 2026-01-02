// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package mirror

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/tidwall/gjson"
)

// Mirror copies data from source to destination using gjson path tags
// Supports nested structs, slices, and maps recursively.
func Mirror(source any, dest any) error {
	// type Dest struct {
	//     Field1 string `mirror:"path.to.field1"`
	//     Field2 int    `mirror:"path.to.field2"`
	//     Nested struct {
	//         SubField string `mirror:"path.to.nested.subfield"`
	//     }
	//     TopLevelField any `mirror:""`
	//     IgnoredField string `mirror:"-"`
	// }
	//
	// The function will read the gjson paths from the `mirror` tags and
	// populate the dest struct fields with values from the source.

	sourceJSON, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("failed to marshal source: %w", err)
	}
	sourceStr := string(sourceJSON)

	destVal := reflect.ValueOf(dest)
	for destVal.Kind() == reflect.Pointer {
		destVal = destVal.Elem()
	}

	return mirrorValue(gjson.Parse(sourceStr), destVal, true)
}

func mirrorValue(source gjson.Result, dest reflect.Value, topLevel bool) error {
	if !dest.IsValid() || !dest.CanSet() {
		return nil
	}

	destType := dest.Type()
	switch dest.Kind() {
	case reflect.Struct:
		return mirrorStruct(source, dest, topLevel)
	case reflect.Slice, reflect.Array:
		return mirrorSlice(source, dest)
	case reflect.Map:
		return mirrorMap(source, dest)
	case reflect.Pointer:
		if dest.IsNil() {
			dest.Set(reflect.New(destType.Elem()))
		}
		return mirrorValue(source, dest.Elem(), topLevel)
	}

	return nil
}

func mirrorStruct(source gjson.Result, dest reflect.Value, topLevel bool) error {
	destType := dest.Type()
	if !topLevel && destType.Name() != "" {
		return nil
	}
	for i := range destType.NumField() {
		source := source
		field := destType.Field(i)
		fieldVal := dest.Field(i)

		if !fieldVal.CanSet() {
			continue // Skip unexported fields
		}

		if path, ok := field.Tag.Lookup("mirror"); ok {
			if path == "-" {
				continue
			}

			if path != "" {
				source = source.Get(path)
			}

			if canSetFieldValue(field.Type) {
				ok, err := setFieldValue(source, fieldVal)
				if err != nil {
					return fmt.Errorf("field %s: %w", field.Name, err)
				}
				if ok {
					continue
				}
			}
		}

		if err := mirrorValue(source, fieldVal, false); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}
	return nil
}

func mirrorSlice(source gjson.Result, destVal reflect.Value) error {
	if !source.Exists() {
		return nil
	}
	if !source.IsArray() {
		return fmt.Errorf("expected JSON array for slice, got: %s", source.Raw)
	}
	array := source.Array()

	if destVal.Kind() == reflect.Slice && destVal.IsNil() {
		destVal.Set(reflect.MakeSlice(destVal.Type(), len(array), len(array)))
	}
	if len(array) != destVal.Len() {
		return fmt.Errorf("mismatched slice lengths: source has %d elements, destination has %d elements", len(array), destVal.Len())
	}
	for i := range destVal.Len() {
		source := array[i]
		if err := mirrorValue(source, destVal.Index(i), false); err != nil {
			return err
		}
	}
	return nil
}

func mirrorMap(source gjson.Result, destVal reflect.Value) error {
	for _, key := range destVal.MapKeys() {
		itemVal := destVal.MapIndex(key)
		if err := mirrorValue(source, itemVal, false); err != nil {
			return err
		}
	}
	return nil
}

func canSetFieldValue(destType reflect.Type) bool {
	for destType.Kind() == reflect.Pointer {
		destType = destType.Elem()
	}
	switch destType.Kind() {
	case reflect.Struct:
		return false
	case reflect.Slice, reflect.Map:
		return canSetFieldValue(destType.Elem())
	default:
		return true
	}
}

func setFieldValue(result gjson.Result, destVal reflect.Value) (bool, error) {
	if !destVal.CanSet() {
		return false, fmt.Errorf("cannot set field value")
	}
	if !result.Exists() {
		return false, nil
	}
	err := json.Unmarshal([]byte(result.Raw), destVal.Addr().Interface())
	if err != nil {
		return false, fmt.Errorf("failed to set field value: %w", err)
	}
	return true, nil
}
