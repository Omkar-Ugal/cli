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
	field, err := fieldFromStruct("", reflect.ValueOf(s))
	if err != nil {
		return nil, err
	}
	return field.Subfields, nil
}

type valueField interface {
	String() string
	Parse(s string) error
}

func fieldFromStruct(pkgPath string, v reflect.Value) (field *Field, err error) {
	s := v
	if s.Kind() == reflect.Pointer {
		if s.IsNil() {
			v2 := reflect.New(s.Type().Elem())
			s = v2
		}
		s = s.Elem()
	}
	if s.Kind() != reflect.Struct {
		return nil, nil
	}
	t := s.Type()

	if pkgPath == "" {
		pkgPath = s.Type().PkgPath()
	}
	if t.PkgPath() != "" && t.PkgPath() != pkgPath {
		return nil, nil
	}

	var fields []Field
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		fieldVal := s.Field(i)

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
		switch {
		case slices.Contains(opts, "hidden"):
			result.Verbosity = FieldVerbosityHidden
		case slices.Contains(opts, "short"):
			result.Verbosity = FieldVerbosityShort
		case slices.Contains(opts, "long"):
			result.Verbosity = FieldVerbosityLong
		default:
			result.Verbosity = FieldVerbosityHidden
		}

		newField, err := fieldFromStruct(pkgPath, fieldVal)
		if err != nil {
			return nil, err
		}
		if newField != nil {
			result.Value = newField.Value
			result.Subfields = newField.Subfields
			result.Verbosity = max(result.Verbosity, newField.Verbosity)
		}

		newField, err = fieldFromSlice(pkgPath, fieldVal)
		if err != nil {
			return nil, err
		}
		if newField != nil {
			result.Value = newField.Value
			result.Elem = newField.Elem
			result.Subfields = newField.Subfields
			result.Verbosity = max(result.Verbosity, newField.Verbosity)
		}

		fields = append(fields, result)
	}

	var value any
	if valueField, ok := v.Interface().(valueField); ok {
		value = valueField
	}

	verbosity := FieldVerbosity(0)
	for _, f := range fields {
		verbosity = max(verbosity, f.Verbosity)
	}
	return &Field{
		Value:     value,
		Subfields: fields,
		Verbosity: verbosity,
	}, nil
}

func fieldFromSlice(pkgPath string, v reflect.Value) (field *Field, err error) {
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
	elem, err := fieldFromStruct(pkgPath, elemVal)
	if err != nil {
		return nil, err
	}
	if elem == nil {
		return nil, nil
	}

	var fields []Field
	for i := range v.Len() {
		vv := v.Index(i)
		field, err := fieldFromStruct(pkgPath, vv)
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
