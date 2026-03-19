// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldsFromStruct_Values(t *testing.T) {
	type Simple struct {
		Name  string `field:",short"`
		Count int    `field:",long"`
	}

	s := Simple{Name: "test", Count: 42}
	fields, err := FieldsFromStruct(s)
	require.NoError(t, err)
	require.Len(t, fields, 2)

	tests := []struct {
		path  string
		value any
	}{
		{"name", "test"},
		{"count", 42},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetFieldByPathString(fields, tt.path)
			require.Len(t, result, 1)
			assert.Equal(t, tt.value, result[0].Value)
		})
	}
}

func TestFieldsFromStruct_NestedValues(t *testing.T) {
	type Inner struct {
		Foo string `field:",short"`
		Bar int    `field:",long"`
	}
	type Outer struct {
		Name   string `field:",short"`
		Nested Inner  `field:",embed"`
	}

	s := Outer{
		Name: "outer",
		Nested: Inner{
			Foo: "hello",
			Bar: 123,
		},
	}
	fields, err := FieldsFromStruct(s)
	require.NoError(t, err)
	require.Len(t, fields, 2)

	tests := []struct {
		path  string
		value any
	}{
		{"name", "outer"},
		{"nested.foo", "hello"},
		{"nested.bar", 123},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetFieldByPathString(fields, tt.path)
			require.Len(t, result, 1)
			assert.Equal(t, tt.value, result[0].Value)
		})
	}
}

func TestFieldsFromStruct_Verbosity(t *testing.T) {
	type AllVerbosities struct {
		ShortField     string `field:",short"`
		LongField      string `field:",long"`
		HiddenField    string `field:",hidden"`
		InvisibleField string `field:",invisible"`
		DefaultField   string `field:""`
	}

	s := AllVerbosities{
		ShortField:     "s",
		LongField:      "l",
		HiddenField:    "h",
		InvisibleField: "i",
		DefaultField:   "d",
	}
	fields, err := FieldsFromStruct(s)
	require.NoError(t, err)
	require.Len(t, fields, 5)

	tests := []struct {
		path      string
		verbosity FieldVerbosity
	}{
		{"short-field", FieldVerbosityShort},
		{"long-field", FieldVerbosityLong},
		{"hidden-field", FieldVerbosityHidden},
		{"invisible-field", FieldVerbosityInvisible},
		// Default verbosity is hidden when no verbosity tag is specified
		{"default-field", FieldVerbosityHidden},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetFieldByPathString(fields, tt.path)
			require.Len(t, result, 1)
			assert.Equal(t, tt.verbosity, result[0].Verbosity)
		})
	}
}

func TestFieldsFromStruct_PatchParsing(t *testing.T) {
	t.Run("create patch set", func(t *testing.T) {
		type WithCreate struct {
			Name string `field:",short" create:"set"`
		}

		s := WithCreate{Name: "test-name"}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "name")
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Create)
		assert.Equal(t, "test-name", result[0].Create.Set)
		assert.Nil(t, result[0].Create.Add)
		assert.Nil(t, result[0].Create.Del)
		assert.False(t, result[0].Create.Required)
	})

	t.Run("edit patch set", func(t *testing.T) {
		type WithEdit struct {
			Count int `field:",short" edit:"set"`
		}

		s := WithEdit{Count: 99}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "count")
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Edit)
		assert.Equal(t, 99, result[0].Edit.Set)
		assert.Nil(t, result[0].Edit.Add)
		assert.Nil(t, result[0].Edit.Del)
	})

	t.Run("edit patch set with create", func(t *testing.T) {
		type WithEditAndCreate struct {
			Count int `field:",short" create:"set" edit:"set"`
		}

		s := WithEditAndCreate{Count: 99}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "count")
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Edit)
		assert.Equal(t, 99, result[0].Edit.Set)
		assert.Nil(t, result[0].Edit.Add)
		assert.Nil(t, result[0].Edit.Del)
	})

	t.Run("create and edit patches", func(t *testing.T) {
		type WithBoth struct {
			Value string `field:",short" create:"set" edit:"set"`
		}

		s := WithBoth{Value: "both"}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "value")
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Create)
		require.NotNil(t, result[0].Edit)
		assert.Equal(t, "both", result[0].Create.Set)
		assert.Equal(t, "both", result[0].Edit.Set)
	})

	t.Run("required flag", func(t *testing.T) {
		type WithRequired struct {
			Name string `field:",short" create:"set,required"`
		}

		s := WithRequired{Name: "required-name"}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "name")
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Create)
		assert.Equal(t, "required-name", result[0].Create.Set)
		assert.True(t, result[0].Create.Required)
	})

	t.Run("no patch when tag not specified", func(t *testing.T) {
		type NoPatch struct {
			Name string `field:",short"`
		}

		s := NoPatch{Name: "no-patch"}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "name")
		require.Len(t, result, 1)
		assert.Nil(t, result[0].Create)
		assert.Nil(t, result[0].Edit)
	})

	t.Run("map del with keys", func(t *testing.T) {
		type WithMap struct {
			Env map[string]string `field:",short" create:"set" edit:"set,del=keys"`
		}

		s := WithMap{Env: map[string]string{"KEY1": "val1", "KEY2": "val2"}}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "env")
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Edit)
		assert.Equal(t, map[string]string{"KEY1": "val1", "KEY2": "val2"}, result[0].Edit.Set)
		// Del is an empty []string (keys type from map[string]string)
		delKeys, ok := result[0].Edit.Del.([]string)
		require.True(t, ok)
		assert.Empty(t, delKeys)
	})

	t.Run("add patch", func(t *testing.T) {
		type WithAdd struct {
			Tags []string `field:",short" edit:"add"`
		}

		s := WithAdd{Tags: []string{"tag1", "tag2"}}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "tags")
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Edit)
		assert.Nil(t, result[0].Edit.Set)
		// Add should be a zero-value slice - the type indicates add is supported but no values yet
		assert.IsType(t, []string(nil), result[0].Edit.Add)
		assert.Empty(t, result[0].Edit.Add)
		assert.Nil(t, result[0].Edit.Del)
	})

	t.Run("del patch", func(t *testing.T) {
		type WithDel struct {
			Tags []string `field:",short" edit:"del"`
		}

		s := WithDel{Tags: []string{"remove1", "remove2"}}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "tags")
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Edit)
		assert.Nil(t, result[0].Edit.Set)
		assert.Nil(t, result[0].Edit.Add)
		// Del should be a zero-value slice - the type indicates del is supported but no values yet
		assert.IsType(t, []string(nil), result[0].Edit.Del)
		assert.Empty(t, result[0].Edit.Del)
	})

	t.Run("add and del patches combined", func(t *testing.T) {
		type WithAddDel struct {
			Add []string `field:",short" edit:"add"`
			Del []string `field:",short" edit:"del"`
		}

		s := WithAddDel{
			Add: []string{"new1", "new2"},
			Del: []string{"old1", "old2"},
		}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		addResult := GetFieldByPathString(fields, "add")
		require.Len(t, addResult, 1)
		require.NotNil(t, addResult[0].Edit)
		assert.Empty(t, addResult[0].Edit.Add)

		delResult := GetFieldByPathString(fields, "del")
		require.Len(t, delResult, 1)
		require.NotNil(t, delResult[0].Edit)
		assert.Empty(t, delResult[0].Edit.Del)
	})

	t.Run("set add del on same field", func(t *testing.T) {
		type WithAll struct {
			Items []string `field:",short" edit:"set,add,del"`
		}

		s := WithAll{Items: []string{"a", "b"}}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "items")
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Edit)
		// Set gets the actual value, but add/del get zero values
		assert.Equal(t, []string{"a", "b"}, result[0].Edit.Set)
		assert.Empty(t, result[0].Edit.Add)
		assert.Empty(t, result[0].Edit.Del)
	})

	t.Run("create with add", func(t *testing.T) {
		type WithCreateAdd struct {
			Labels []string `field:",short" create:"add"`
		}

		s := WithCreateAdd{Labels: []string{"label1"}}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "labels")
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Create)
		assert.Nil(t, result[0].Create.Set)
		assert.Empty(t, result[0].Create.Add)
		assert.Nil(t, result[0].Create.Del)
	})

	t.Run("map with add", func(t *testing.T) {
		type WithMapAdd struct {
			Env map[string]string `field:",short" edit:"add"`
		}

		s := WithMapAdd{Env: map[string]string{"KEY1": "val1", "KEY2": "val2"}}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "env")
		require.Len(t, result, 1)
		require.NotNil(t, result[0].Edit)
		// Add is an empty map (allocated, not nil)
		addMap, ok := result[0].Edit.Add.(map[string]string)
		require.True(t, ok)
		assert.Empty(t, addMap)
	})
}

func TestFieldsFromStruct_Slices(t *testing.T) {
	type Item struct {
		Name  string `field:",short"`
		Value int    `field:",long"`
	}
	type WithSlice struct {
		Items []Item `field:",embed"`
	}

	s := WithSlice{
		Items: []Item{
			{Name: "first", Value: 1},
			{Name: "second", Value: 2},
		},
	}
	fields, err := FieldsFromStruct(s)
	require.NoError(t, err)
	require.Len(t, fields, 1)

	itemsField := GetFieldByPathString(fields, "items")
	require.Len(t, itemsField, 1)
	require.NotNil(t, itemsField[0].Elem)
	require.Len(t, itemsField[0].Subfields, 2)

	tests := []struct {
		path  string
		value any
	}{
		{"items.0.name", "first"},
		{"items.0.value", 1},
		{"items.1.name", "second"},
		{"items.1.value", 2},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetFieldByPathString(fields, tt.path)
			require.Len(t, result, 1)
			assert.Equal(t, tt.value, result[0].Value)
		})
	}
}

func TestFieldsFromStruct_PointerFields(t *testing.T) {
	type Inner struct {
		Name string `field:",short"`
	}
	type WithPointer struct {
		Inner *Inner `field:",embed"`
	}

	t.Run("non-nil pointer", func(t *testing.T) {
		s := WithPointer{Inner: &Inner{Name: "ptr-value"}}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "inner.name")
		require.Len(t, result, 1)
		assert.Equal(t, "ptr-value", result[0].Value)
	})

	t.Run("nil pointer", func(t *testing.T) {
		s := WithPointer{Inner: nil}
		fields, err := FieldsFromStruct(s)
		require.NoError(t, err)

		result := GetFieldByPathString(fields, "inner.name")
		require.Len(t, result, 1)
		assert.Empty(t, result[0].Value)
	})
}

func TestFieldsFromStruct_CustomFieldName(t *testing.T) {
	type CustomNames struct {
		MyField    string `field:"custom-name,short"`
		OtherField string `field:"other,long"`
	}

	s := CustomNames{MyField: "value1", OtherField: "value2"}
	fields, err := FieldsFromStruct(s)
	require.NoError(t, err)
	require.Len(t, fields, 2)

	tests := []struct {
		path  string
		value any
	}{
		{"custom-name", "value1"},
		{"other", "value2"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetFieldByPathString(fields, tt.path)
			require.Len(t, result, 1)
			assert.Equal(t, tt.value, result[0].Value)
		})
	}
}

func TestFieldsFromStruct_SkipField(t *testing.T) {
	type WithSkip struct {
		Name    string `field:",short"`
		Skipped string `field:"-"`
	}

	s := WithSkip{Name: "included", Skipped: "should-not-appear"}
	fields, err := FieldsFromStruct(s)
	require.NoError(t, err)
	require.Len(t, fields, 1)

	tests := []struct {
		path     string
		value    any
		expected bool
	}{
		{"name", "included", true},
		{"skipped", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetFieldByPathString(fields, tt.path)
			if tt.expected {
				require.Len(t, result, 1)
				assert.Equal(t, tt.value, result[0].Value)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestFieldsFromStruct_UnexportedFieldsIgnored(t *testing.T) {
	type WithUnexported struct {
		Public  string `field:",short"`
		private string `field:",short"`
	}

	s := WithUnexported{Public: "visible", private: "invisible"}
	fields, err := FieldsFromStruct(s)
	require.NoError(t, err)
	require.Len(t, fields, 1)

	tests := []struct {
		path     string
		value    any
		expected bool
	}{
		{"public", "visible", true},
		{"private", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetFieldByPathString(fields, tt.path)
			if tt.expected {
				require.Len(t, result, 1)
				assert.Equal(t, tt.value, result[0].Value)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}
