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
	"time"

	"github.com/ettle/strcase"
	"github.com/mitchellh/mapstructure"
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

type ValueField interface {
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

		parsedField := ParseField(field)
		if parsedField == nil {
			continue
		}
		result := Field{
			Name:      parsedField.Name,
			Verbosity: parsedField.Verbosity,
			Value:     fieldVal.Interface(),
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
	if ValueField, ok := v.Interface().(ValueField); ok {
		value = ValueField
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

type ParsedField struct {
	Name      string
	Verbosity FieldVerbosity
}

func ParseField(field reflect.StructField) *ParsedField {
	if !field.IsExported() {
		return nil
	}
	tag := field.Tag.Get("field")
	if tag == "-" {
		return nil
	}

	opts := strings.Split(tag, ",")
	name, opts := opts[0], opts[1:]
	if name == "" {
		name = field.Name
		name = strcase.ToKebab(name)
	}

	var verbosity FieldVerbosity
	switch {
	case slices.Contains(opts, "invisible"):
		verbosity = FieldVerbosityInvisible
	case slices.Contains(opts, "hidden"):
		verbosity = FieldVerbosityHidden
	case slices.Contains(opts, "short"):
		verbosity = FieldVerbosityShort
	case slices.Contains(opts, "long"):
		verbosity = FieldVerbosityLong
	default:
		verbosity = FieldVerbosityHidden
	}

	return &ParsedField{
		Name:      name,
		Verbosity: verbosity,
	}
}

// HACK: avoid use of this method, and prefer using the info available directly
// on the Field - this function makes heavy assumptions about the structure of
// field data and how values are read/written. Currently it is only used for
// visual editing.
func DecodeStruct(input any, output any) error {
	config := mapstructure.DecoderConfig{
		TagName:     "field",
		ErrorUnused: true,
		Result:      output,
		// TODO: more hooks probably needed
		DecodeHook: mapstructure.StringToTimeHookFunc(time.RFC3339),
	}
	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}
