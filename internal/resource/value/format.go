// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package value

import (
	"cmp"
	"encoding"
	"fmt"
	"reflect"
	"strings"
)

type Wrapped interface {
	Unwrap() any
}

func Format(value any) (string, error) {
	for {
		unwrapped, ok := value.(Wrapped)
		if !ok {
			break
		}
		value = unwrapped.Unwrap()
	}
	if value, ok := value.(fmt.Stringer); ok {
		return value.String(), nil
	}
	if value, ok := value.(encoding.TextMarshaler); ok {
		dt, err := value.MarshalText()
		return string(dt), err
	}

	if value == nil {
		return "<nil>", nil
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return "<nil>", nil
		}
		return Format(v.Elem().Interface())
	case reflect.String:
		return v.String(), nil
	case reflect.Slice, reflect.Array:
		var result []string
		for i := range v.Len() {
			val := v.Index(i)
			valStr, err := Format(val.Interface())
			if err != nil {
				return "", err
			}
			result = append(result, valStr)
		}
		return strings.Join(result, ","), nil
	case reflect.Map:
		var result []string
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			keyStr, err := Format(key.Interface())
			if err != nil {
				return "", err
			}
			valStr, err := Format(val.Interface())
			if err != nil {
				return "", err
			}
			result = append(result, fmt.Sprintf("%s=%s", keyStr, valStr))
		}
		return strings.Join(result, ","), nil
	case reflect.Struct:
		var result []string
		for i := range v.NumField() {
			field := v.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			val := v.Field(i)
			valStr, err := Format(val.Interface())
			if err != nil {
				return "", err
			}
			if valStr == "" {
				continue
			}
			name := cmp.Or(
				strings.SplitN(field.Tag.Get("field"), ",", 2)[0],
				strings.ToLower(field.Name),
			)
			result = append(result, fmt.Sprintf("%s=%s", name, valStr))
		}
		return strings.Join(result, ","), nil
	default:
		return fmt.Sprintf("%v", value), nil
	}
}
