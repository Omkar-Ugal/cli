// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/charmbracelet/x/ansi"

	"unikraft.com/cli/internal/resource"
	"unikraft.com/cli/internal/resource/value"
	"unikraft.com/x/filters"
)

type fieldAdaptor struct {
	children []resource.Field
	field    *resource.Field
	entries  []string
	sliceVal *string
}

func newFieldAdaptor(fields []resource.Field) filters.Adaptor {
	return &fieldAdaptor{children: fields}
}

func (a *fieldAdaptor) Select(key []string) (filters.Adaptor, bool) {
	if a.sliceVal != nil {
		return nil, false
	}
	matched := resource.GetFieldByPath(a.children, key)
	if len(matched) == 0 {
		if len(key) >= 1 {
			parentKey := key[:len(key)-1]
			var parent *resource.Field
			if len(parentKey) == 0 {
				parent = a.field
			} else if pm := resource.GetFieldByPath(a.children, parentKey); len(pm) == 1 {
				parent = &pm[0]
			}
			if parent != nil {
				if slice, sok := getSliceValue(parent.Value); sok {
					idx, err := strconv.Atoi(key[len(key)-1])
					if err == nil && idx >= 0 && idx < len(slice) {
						s := slice[idx]
						return &fieldAdaptor{sliceVal: &s}, true
					}
				}
			}
		}
		return nil, false
	}

	if len(matched) == 1 {
		f := matched[0]
		if len(f.Subfields) > 0 {
			names := make([]string, len(f.Subfields))
			for i, sub := range f.Subfields {
				names[i] = sub.Name
			}
			return &fieldAdaptor{children: f.Subfields, field: &f, entries: names}, true
		}
		if slice, sok := getSliceValue(f.Value); sok {
			names := make([]string, len(slice))
			for i := range slice {
				names[i] = strconv.Itoa(i)
			}
			return &fieldAdaptor{field: &f, entries: names}, true
		}
		return &fieldAdaptor{field: &f}, true
	}

	names := make([]string, len(matched))
	for i, f := range matched {
		names[i] = f.Name
	}
	return &fieldAdaptor{entries: names}, true
}

func (a *fieldAdaptor) String() string {
	if a.sliceVal != nil {
		return *a.sliceVal
	}
	if a.field == nil {
		return ""
	}
	out, _ := a.field.Render(value.RenderOpts{})
	return ansi.Strip(out)
}

func (a *fieldAdaptor) Value() any {
	if a.sliceVal != nil {
		return *a.sliceVal
	}
	if a.field == nil {
		return nil
	}
	return a.field.Value
}

func (a *fieldAdaptor) Entries() []string {
	return a.entries
}

func (a *fieldAdaptor) Compare(other string) (int, bool) {
	if a.sliceVal != nil {
		return 0, false
	}
	if a.field == nil || a.field.Value == nil {
		return 0, false
	}
	if !isOrderable(a.field.Value) {
		return 0, false
	}
	parsed, err := value.ParseNew([]string{other}, a.field.Value)
	if err != nil {
		return 0, false
	}
	return value.Compare(a.field.Value, parsed), true
}

func isOrderable(v any) bool {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool, reflect.String:
		return true
	}
	return rv.Type().ConvertibleTo(reflect.TypeFor[time.Time]())
}

func getSliceValue(value any) ([]string, bool) {
	if value == nil {
		return nil, false
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice {
		return nil, false
	}
	result := make([]string, rv.Len())
	for i := range rv.Len() {
		elem := rv.Index(i)
		if s, ok := elem.Interface().(string); ok {
			result[i] = s
		} else if s, ok := elem.Interface().(fmt.Stringer); ok {
			result[i] = s.String()
		} else {
			result[i] = fmt.Sprintf("%v", elem.Interface())
		}
	}
	return result, true
}
