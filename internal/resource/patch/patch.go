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
	"reflect"
	"slices"
	"strconv"
	"strings"

	"unikraft.com/cli/internal/resource"
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
			original = field.Patch
			field.Patch = nil
			patch = &field.Patch
		}
		if original == nil {
			original = &resource.Patch{}
		}

		done := false
		if value, ok := spec.Set[keyStr]; ok {
			done = true
			if original.Set != nil {
				set, err := parseNewValue(value, original.Set)
				if err != nil {
					return nil, fmt.Errorf("failed to unpack set value for %s: %w", keyStr, err)
				}
				*patch = &resource.Patch{Set: set}
			} else {
				setForbiddenFields[keyStr] = struct{}{}
			}
		}
		if value, ok := spec.Add[keyStr]; ok {
			done = true
			if original.Add != nil {
				add, err := parseNewValue(value, original.Add)
				if err != nil {
					return nil, fmt.Errorf("failed to unpack add value for %s: %w", keyStr, err)
				}
				*patch = &resource.Patch{Add: add}
			} else {
				addForbiddenFields[keyStr] = struct{}{}
			}
		}
		if value, ok := spec.Del[keyStr]; ok {
			done = true
			if original.Del != nil {
				del, err := parseNewValue(value, original.Del)
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

func parseNewValue(input []string, output any) (any, error) {
	parsedVal, err := parseNewReflect(input, reflect.ValueOf(output))
	if err != nil {
		return nil, err
	}
	return parsedVal.Interface(), nil
}

func parseNewReflect(input []string, output reflect.Value) (reflect.Value, error) {
	newVal := reflect.New(output.Type())
	err := parseReflect(input, newVal.Elem())
	if err != nil {
		return reflect.Value{}, err
	}
	return newVal.Elem(), nil
}

func parseReflect(input []string, value reflect.Value) error {
	if len(input) == 0 {
		return nil
	}

	output := value
	for output.Kind() == reflect.Pointer {
		if output.IsNil() {
			output.Set(reflect.New(output.Type().Elem()))
		}
		output = output.Elem()
	}

	switch output.Kind() {
	case reflect.Pointer:
		if output.IsNil() {
			output.Set(reflect.New(output.Type().Elem()))
		}
		return parseReflect(input, output.Elem())
	case reflect.String:
		output.SetString(input[0])
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(input[0], 10, output.Type().Bits())
		if err != nil {
			return err
		}
		output.SetInt(v)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(input[0], 10, output.Type().Bits())
		if err != nil {
			return err
		}
		output.SetUint(v)
		return nil
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(input[0], output.Type().Bits())
		if err != nil {
			return err
		}
		output.SetFloat(v)
		return nil
	case reflect.Bool:
		v, err := strconv.ParseBool(input[0])
		if err != nil {
			return err
		}
		output.SetBool(v)
		return nil
	case reflect.Slice:
		slice := reflect.MakeSlice(output.Type(), 0, 0)
		for _, input := range input {
			for item := range strings.SplitSeq(input, ",") {
				val := reflect.New(output.Type().Elem()).Elem()
				err := parseReflect([]string{item}, val)
				if err != nil {
					return err
				}
				slice = reflect.Append(slice, val)
			}
		}
		output.Set(slice)
		return nil
	case reflect.Map:
		mapp := reflect.MakeMap(output.Type())
		for _, input := range input {
			for item := range strings.SplitSeq(input, ",") {
				if item == "" {
					continue
				}
				k, v, _ := strings.Cut(item, "=")
				key := reflect.New(output.Type().Key()).Elem()
				err := parseReflect([]string{k}, key)
				if err != nil {
					return err
				}
				val := reflect.New(output.Type().Elem()).Elem()
				err = parseReflect([]string{v}, val)
				if err != nil {
					return err
				}
				mapp.SetMapIndex(key, val)
			}
		}
		output.Set(mapp)
		return nil
	case reflect.Struct:
		valueField, ok := value.Interface().(resource.ValueField)
		if ok {
			return valueField.Parse(input[0])
		}

		kv := map[string]string{}
		fields := strings.Split(input[0], ",")
		for _, field := range fields {
			field = strings.TrimSpace(field)
			k, v, _ := strings.Cut(field, "=")
			kv[k] = v
		}
		return resource.DecodeStruct(kv, value.Addr().Interface())
	default:
		return fmt.Errorf("unsupported type: %T", value.Interface())
	}
}
