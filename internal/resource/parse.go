// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func parseNewValue(input string, output any) (any, error) {
	parsedVal, err := parseNewReflect(input, reflect.ValueOf(output))
	if err != nil {
		return nil, err
	}
	return parsedVal.Interface(), nil
}

func parseNewReflect(input string, output reflect.Value) (reflect.Value, error) {
	newVal := reflect.New(output.Type())
	err := parseReflect(input, newVal.Elem())
	if err != nil {
		return reflect.Value{}, err
	}
	return newVal.Elem(), nil
}

func parseReflect(input string, output reflect.Value) error {
	switch output.Kind() {
	case reflect.Pointer:
		if output.IsNil() {
			output.Set(reflect.New(output.Type().Elem()))
		}
		return parseReflect(input, output.Elem())
	case reflect.String:
		output.SetString(input)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(input, 10, output.Type().Bits())
		if err != nil {
			return err
		}
		output.SetInt(v)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(input, 10, output.Type().Bits())
		if err != nil {
			return err
		}
		output.SetUint(v)
		return nil
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(input, output.Type().Bits())
		if err != nil {
			return err
		}
		output.SetFloat(v)
		return nil
	case reflect.Bool:
		v, err := strconv.ParseBool(input)
		if err != nil {
			return err
		}
		output.SetBool(v)
		return nil
	case reflect.Slice:
		inputs := strings.Split(input, ",")
		slice := reflect.MakeSlice(output.Type(), len(inputs), len(inputs))
		for i, item := range inputs {
			err := parseReflect(item, slice.Index(i))
			if err != nil {
				return err
			}
		}
		output.Set(slice)
		return nil
	case reflect.Map:
		inputs := strings.Split(input, ",")
		mapp := reflect.MakeMap(output.Type())
		for _, item := range inputs {
			if item == "" {
				continue
			}
			k, v, _ := strings.Cut(item, "=")
			key := reflect.New(output.Type().Key()).Elem()
			err := parseReflect(k, key)
			if err != nil {
				return err
			}
			val := reflect.New(output.Type().Elem()).Elem()
			err = parseReflect(v, val)
			if err != nil {
				return err
			}
			mapp.SetMapIndex(key, val)
		}
		output.Set(mapp)
		return nil
	default:
		return fmt.Errorf("unsupported kind: %s", output.Kind())
	}
}
