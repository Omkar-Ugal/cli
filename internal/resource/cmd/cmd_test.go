// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/mitchellh/copystructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/resource"
)

func TestCommands(t *testing.T) {
	ctx := context.Background()
	sandbox := &resource.Sandbox{}

	baseTestStore := map[string]resource.Test{
		"test1": {
			Name: "test1",
			URL:  "https://example.com",
			Settings: resource.TestSettings{
				X: 42,
				Y: "hello",
			},
			Authors: []resource.TestAuthor{
				{Name: "Alice", Email: "alice@example.com"},
				{Name: "Bob", Email: "bob@example.com"},
			},
		},
		"test2": {
			Name: "test2",
			URL:  "https://example.org",
			Settings: resource.TestSettings{
				X: 7,
				Y: "world",
			},
			Authors: []resource.TestAuthor{
				{Name: "Charlie", Email: "charlie@example.com"},
				{Name: "Dana", Email: "dana@example.com"},
			},
		},
	}

	t.Run("list", func(t *testing.T) {
		cloned, err := copystructure.Copy(baseTestStore)
		require.NoError(t, err)
		resource.TestStore = cloned.(map[string]resource.Test)

		var empty resource.Test
		resources, err := empty.List(ctx)
		require.NoError(t, err)
		assert.Len(t, resources, 2)

		var listOut bytes.Buffer
		listCmd := &ResourceListCmd[resource.Test]{}
		err = listCmd.Run(ctx, testConfig(&listOut), sandbox)
		require.NoError(t, err)

		output := listOut.String()
		assert.Contains(t, output, "test1")
		assert.Contains(t, output, "test2")
	})

	t.Run("get", func(t *testing.T) {
		cloned, err := copystructure.Copy(baseTestStore)
		require.NoError(t, err)
		resource.TestStore = cloned.(map[string]resource.Test)

		var empty resource.Test
		resources, err := empty.Get(ctx, []string{"test1"})
		require.NoError(t, err)
		require.Len(t, resources, 1)

		test := resources[0].(resource.Test)
		assert.Equal(t, "test1", test.Name)
		assert.Equal(t, 42, test.Settings.X)
		assert.Equal(t, "hello", test.Settings.Y)

		fields, err := test.Fields()
		require.NoError(t, err)
		assert.NotEmpty(t, fields)

		var inspectOut bytes.Buffer
		inspectCmd := &ResourceInspectCmd[resource.Test]{
			Name: []string{"test1"},
		}
		err = inspectCmd.Run(ctx, testConfig(&inspectOut), sandbox)
		require.NoError(t, err)

		output := inspectOut.String()
		assert.Contains(t, output, "test1")
		assert.Contains(t, output, "https://example.com")
	})

	t.Run("create", func(t *testing.T) {
		resource.TestStore = map[string]resource.Test{}

		var empty resource.Test
		templateFields, err := empty.Fields()
		require.NoError(t, err)

		for key, field := range resource.IterFields(templateFields) {
			if field.Create == nil {
				continue
			}
			switch key.String() {
			case "name":
				field.Create.Set = "test-new"
			case "settings.x":
				field.Create.Set = 100
			case "settings.y":
				field.Create.Set = "created"
			}
		}

		res, err := empty.Create(ctx, templateFields)
		require.NoError(t, err)

		created := res.(resource.Test)
		assert.Equal(t, "test-new", created.Name)
		assert.Equal(t, 100, created.Settings.X)
		assert.Equal(t, "created", created.Settings.Y)
		assert.Contains(t, resource.TestStore, "test-new")
	})

	t.Run("edit", func(t *testing.T) {
		editStore := map[string]resource.Test{
			"test-edit": {
				Name: "test-edit",
				URL:  "https://example.com",
				Settings: resource.TestSettings{
					X: 10,
					Y: "original",
				},
			},
		}
		cloned, err := copystructure.Copy(editStore)
		require.NoError(t, err)
		resource.TestStore = cloned.(map[string]resource.Test)

		var empty resource.Test
		resources, err := empty.Get(ctx, []string{"test-edit"})
		require.NoError(t, err)
		require.Len(t, resources, 1)

		target := resources[0]
		templateFields, err := target.Fields()
		require.NoError(t, err)

		for key, field := range resource.IterFields(templateFields) {
			if field.Patch == nil {
				continue
			}
			switch key.String() {
			case "settings.x":
				field.Patch.Set = 999
			case "settings.y":
				field.Patch.Set = "modified"
			}
		}

		res, err := empty.Edit(ctx, target, templateFields)
		require.NoError(t, err)

		edited := res.(resource.Test)
		assert.Equal(t, "test-edit", edited.Name)
		assert.Equal(t, 999, edited.Settings.X)
		assert.Equal(t, "modified", edited.Settings.Y)

		stored := resource.TestStore["test-edit"]
		assert.Equal(t, 999, stored.Settings.X)
		assert.Equal(t, "modified", stored.Settings.Y)
	})

	t.Run("delete", func(t *testing.T) {
		deleteStore := map[string]resource.Test{
			"test-delete": {
				Name: "test-delete",
				URL:  "https://example.com",
			},
			"test-keep": {
				Name: "test-keep",
				URL:  "https://example.org",
			},
		}
		cloned, err := copystructure.Copy(deleteStore)
		require.NoError(t, err)
		resource.TestStore = cloned.(map[string]resource.Test)

		var empty resource.Test
		resources, err := empty.Get(ctx, []string{"test-delete"})
		require.NoError(t, err)
		require.Len(t, resources, 1)

		err = empty.Delete(ctx, resources)
		require.NoError(t, err)

		assert.NotContains(t, resource.TestStore, "test-delete")
		assert.Contains(t, resource.TestStore, "test-keep")

		resources, err = empty.Get(ctx, []string{"test-delete"})
		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}

func testConfig(out io.Writer) *config.Config {
	return &config.Config{
		Context: context.Background(),
		Stdin:   &bytes.Buffer{},
		Stdout:  out,
		Stderr:  out,
	}
}
