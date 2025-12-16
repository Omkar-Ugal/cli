// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"reflect"
	"slices"
	"strings"

	"github.com/ettle/strcase"
)

func FieldsFromStruct(s any) (fields []Field, err error) {
	reflectVal := reflect.ValueOf(s)
	if reflectVal.Kind() == reflect.Pointer {
		reflectVal = reflectVal.Elem()
	}
	if reflectVal.Kind() != reflect.Struct {
		return nil, nil
	}
	reflectType := reflectVal.Type()

	for i := range reflectType.NumField() {
		field := reflectType.Field(i)
		if !field.IsExported() {
			continue
		}
		fieldVal := reflectVal.Field(i).Interface()

		name := field.Tag.Get("field")
		if name == "-" {
			continue
		}
		opts := strings.Split(name, ",")
		name, opts = opts[0], opts[1:]
		if name == "" {
			name = field.Name
			name = strcase.ToKebab(name)
		}
		result := Field{
			Name:  name,
			Value: fieldVal,
		}
		if slices.Contains(opts, "short") {
			result.Verbosity = max(result.Verbosity, FieldVerbosityShort)
		}
		if slices.Contains(opts, "long") {
			result.Verbosity = max(result.Verbosity, FieldVerbosityLong)
		}
		if field.Anonymous {
			subfields, err := FieldsFromStruct(fieldVal)
			if err != nil {
				return nil, err
			}
			result.Subfields = subfields
			if len(result.Subfields) > 0 {
				result.Value = nil
				for _, sf := range result.Subfields {
					result.Verbosity = max(result.Verbosity, sf.Verbosity)
				}
			}
		}

		fields = append(fields, result)
	}

	return fields, nil
}
