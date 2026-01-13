// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

type Type struct {
	Name  string
	Names string
}

type Resource interface {
	Type() Type
	Key() string
	Fields() ([]Field, error)
	Raw() any
}

type GettableResource interface {
	Resource
	Get(ctx context.Context, keys []string) ([]Resource, error)
	List(ctx context.Context) ([]Resource, error)
}

type EditableResource interface {
	GettableResource
	Edit(ctx context.Context, target Resource, fields []Field) (Resource, error)
}

type CreatableResource interface {
	GettableResource
	Create(ctx context.Context, fields []Field) (Resource, error)
}

type DeletableResource interface {
	GettableResource
	Delete(ctx context.Context, targets []Resource) error
}

type Link struct {
	Type string
	Key  string
}

type Field struct {
	// Name is the name of the field
	Name string `json:"name"`
	// Value contains the value of the field
	Value any `json:"value,omitempty"`

	// Subfields allow defining nested structures
	Subfields []Field `json:"subfields,omitempty"`
	// Elem is used to indicate that all subfields have the same substructure
	// (e.g. for arrays)
	Elem *Field `json:"elem,omitempty"`

	Links []Link `json:"links,omitempty"`

	// display settings
	Verbosity FieldVerbosity `json:"verbosity"`
	Hyperlink string         `json:"hyperlink,omitempty"`

	// settings for creating or patching resources
	Create *Patch `json:"create,omitempty"`
	Patch  *Patch `json:"patch,omitempty"`
}

func (f Field) HasChildren() bool {
	return len(f.Subfields) > 0 || f.Elem != nil
}

func (f Field) IsEmpty() bool {
	if f.Value != nil {
		return reflect.ValueOf(f.Value).IsZero()
	}
	if f.Elem != nil {
		return len(f.Subfields) == 0
	}

	for _, subfield := range f.Subfields {
		if !subfield.IsEmpty() {
			return false
		}
	}
	return true
}

func (f Field) Get(name string) (Field, bool) {
	for _, subfield := range f.Subfields {
		if subfield.Name == name {
			return subfield, true
		}
	}
	return Field{}, false
}

type Patch struct {
	Set any `json:"set,omitempty"`
	Add any `json:"add,omitempty"`
	Del any `json:"del,omitempty"`

	// Required indicates whether a field must be provided when patching a resource.
	Required bool `json:"required,omitempty"`
}

func (f Field) ValueString() string {
	return valueString(f.Value)
}

func valueString(value any) string {
	for {
		unwrapped, ok := value.(interface{ Unwrap() any })
		if !ok {
			break
		}
		value = unwrapped.Unwrap()
	}
	if value, ok := value.(fmt.Stringer); ok {
		return value.String()
	}

	if value == nil {
		return "<nil>"
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Slice, reflect.Array:
		var result []string
		for i := range v.Len() {
			result = append(result, valueString(v.Index(i).Interface()))
		}
		return "[" + strings.Join(result, " ") + "]"
	case reflect.Map:
		var result []string
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			result = append(result, fmt.Sprintf("%s:%s", valueString(key.Interface()), valueString(val.Interface())))
		}
		return "{" + strings.Join(result, " ") + "}"
	default:
		return fmt.Sprintf("%v", value)
	}
}

type FieldVerbosity int

const (
	FieldVerbosityInvisible FieldVerbosity = iota
	FieldVerbosityHidden
	FieldVerbosityLong
	FieldVerbosityShort
)

func (v FieldVerbosity) String() string {
	if v < FieldVerbosityHidden {
		v = FieldVerbosityHidden
	}
	switch v {
	case FieldVerbosityInvisible:
		return "invisible"
	case FieldVerbosityHidden:
		return "hidden"
	case FieldVerbosityLong:
		return "long"
	case FieldVerbosityShort:
		return "short"
	default:
		return "always"
	}
}

func (v FieldVerbosity) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `%q`, v.String()), nil
}
