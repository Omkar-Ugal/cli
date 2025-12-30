// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"context"
	"fmt"
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

	// TODO: link to other resources, e.g. instance -> image
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

type Field struct {
	Name  string
	Value any

	Hyperlink string

	Create Patch
	Patch  Patch

	Verbosity FieldVerbosity

	Subfields []Field
}

type Patch struct {
	Set any
	Add any
	Del any

	// Required indicates whether a field must be provided when patching a resource.
	Required bool
}

func (f Field) ValueString() string {
	value := f.Value
	if value == nil {
		return ""
	}
	for {
		unwrapped, ok := value.(interface{ Unwrap() any })
		if !ok {
			break
		}
		value = unwrapped.Unwrap()
	}
	if str, ok := value.(fmt.Stringer); ok {
		return str.String()
	}
	return fmt.Sprintf("%v", value)
}

type FieldVerbosity int

const (
	FieldVerbosityHidden FieldVerbosity = iota
	FieldVerbosityLong
	FieldVerbosityShort
)
