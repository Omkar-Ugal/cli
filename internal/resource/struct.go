// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/ettle/strcase"
)

// FieldsFromStruct is a helper that converts a struct into a slice of Fields
// based on the `field` tags defined on the struct's fields.
func FieldsFromStruct(s any) (fields []Field, err error) {
	field, err := fieldFromStruct(reflect.ValueOf(s), true)
	if err != nil {
		return nil, err
	}
	return field.Subfields, nil
}

func fieldFromStruct(v reflect.Value, topLevel bool) (field *Field, err error) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v = reflect.New(v.Type().Elem())
		} else {
			v = v.Elem()
		}
	}
	if v.Kind() != reflect.Struct {
		return nil, nil
	}
	if !topLevel && v.Type().Name() != "" {
		return nil, nil
	}
	t := v.Type()

	var fields []Field
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		fieldVal := v.Field(i)

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
			Value: fieldVal.Interface(),
		}
		if slices.Contains(opts, "short") {
			result.Verbosity = max(result.Verbosity, FieldVerbosityShort)
		}
		if slices.Contains(opts, "long") {
			result.Verbosity = max(result.Verbosity, FieldVerbosityLong)
		}

		newField, err := fieldFromStruct(fieldVal, false)
		if err != nil {
			return nil, err
		}
		if newField != nil {
			result.Value = nil
			result.Subfields = newField.Subfields
			result.Verbosity = max(result.Verbosity, newField.Verbosity)
		}

		newField, err = fieldFromSlice(fieldVal)
		if err != nil {
			return nil, err
		}
		if newField != nil {
			result.Value = nil
			result.Elem = newField.Elem
			result.Subfields = newField.Subfields
			result.Verbosity = max(result.Verbosity, newField.Verbosity)
		}

		fields = append(fields, result)
	}

	verbosity := FieldVerbosityHidden
	for _, f := range fields {
		verbosity = max(verbosity, f.Verbosity)
	}
	return &Field{
		Subfields: fields,
		Verbosity: verbosity,
	}, nil
}

func fieldFromSlice(v reflect.Value) (field *Field, err error) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v = reflect.New(v.Type().Elem())
		} else {
			v = v.Elem()
		}
	}
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, nil
	}

	elemType := v.Type().Elem()
	elemVal := reflect.New(elemType).Elem()
	elem, err := fieldFromStruct(elemVal, false)
	if err != nil {
		return nil, err
	}
	if elem == nil {
		return nil, nil
	}

	var fields []Field
	for i := range v.Len() {
		vv := v.Index(i)
		field, err := fieldFromStruct(vv, false)
		if err != nil {
			return nil, err
		}
		if field == nil {
			continue
		}
		field.Name = strconv.Itoa(i)
		fields = append(fields, *field)
	}

	return &Field{
		Elem:      elem,
		Subfields: fields,
		Verbosity: elem.Verbosity,
	}, nil
}
