// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package mirror

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMirror_BasicExample(t *testing.T) {
	source := map[string]any{
		"path": map[string]any{
			"to": map[string]any{
				"field1": "value1",
				"field2": 42,
				"nested": map[string]any{
					"subfield": "nested_value",
				},
			},
		},
	}

	type Dest struct {
		Field1 string `mirror:"path.to.field1"`
		Field2 int    `mirror:"path.to.field2"`
		Nested struct {
			SubField string `mirror:"path.to.nested.subfield"`
		}
	}

	var dest Dest
	err := Mirror(source, &dest)
	require.NoError(t, err)

	assert.Equal(t, "value1", dest.Field1)
	assert.Equal(t, 42, dest.Field2)
	assert.Equal(t, "nested_value", dest.Nested.SubField)
}

func TestMirror_TopLevelField(t *testing.T) {
	source := map[string]any{
		"name": "John",
		"age":  30,
	}

	type Dest struct {
		Data any `mirror:""`
	}

	var dest Dest
	err := Mirror(source, &dest)
	require.NoError(t, err)

	data, ok := dest.Data.(map[string]any)
	require.True(t, ok, "Data should be map[string]any")

	assert.Equal(t, "John", data["name"])
	assert.Equal(t, float64(30), data["age"])
}

func TestMirror_IgnoredField(t *testing.T) {
	source := map[string]any{
		"field1": "value1",
		"field2": 42,
	}

	type Dest struct {
		Field1       string `mirror:"field1"`
		IgnoredField string `mirror:"-"`
		DefaultValue string
	}

	dest := Dest{
		IgnoredField: "should_not_change",
		DefaultValue: "default",
	}

	err := Mirror(source, &dest)
	require.NoError(t, err)

	assert.Equal(t, "value1", dest.Field1)
	assert.Equal(t, "should_not_change", dest.IgnoredField)
}

func TestMirror_NonExistentPath(t *testing.T) {
	source := map[string]any{
		"field1": "value1",
	}

	type Dest struct {
		Field1 string `mirror:"field1"`
		Field2 string `mirror:"nonexistent.path"`
	}

	var dest Dest
	err := Mirror(source, &dest)
	require.NoError(t, err)

	assert.Equal(t, "value1", dest.Field1)
	assert.Empty(t, dest.Field2)
}

func TestMirror_NonPointerDestination(t *testing.T) {
	source := map[string]any{
		"field": "value",
	}

	type Dest struct {
		Field string `mirror:"field"`
	}

	dest := Dest{}

	// Non-pointer destination should work because Mirror dereferences pointers
	// and the value itself is valid even if not a pointer
	err := Mirror(source, dest)
	require.NoError(t, err)

	// The original dest is not modified because it's passed by value
	assert.Empty(t, dest.Field)
}
