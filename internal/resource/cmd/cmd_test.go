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
	resourcet "unikraft.com/cli/internal/resource/testing"
)

var baseTestStore = map[string]resourcet.TestResource{
	"test1": {
		ID:   "id-test1",
		Name: "test1",
		URL:  "https://example.com",
		Settings: resourcet.TestSettings{
			X: 42,
			Y: "hello",
		},
		Authors: []resourcet.TestAuthor{
			{Name: "Alice", Email: "alice@example.com"},
			{Name: "Bob", Email: "bob@example.com"},
		},
	},
	"test2": {
		ID:   "id-test2",
		Name: "test2",
		URL:  "https://example.org",
		Settings: resourcet.TestSettings{
			X: 7,
			Y: "world",
		},
		Authors: []resourcet.TestAuthor{
			{Name: "Charlie", Email: "charlie@example.com"},
			{Name: "Dana", Email: "dana@example.com"},
		},
	},
}

func TestList(t *testing.T) {
	ctx := context.Background()
	sandbox := &resource.Sandbox{}

	cloned, err := copystructure.Copy(baseTestStore)
	require.NoError(t, err)
	resourcet.TestStore = cloned.(map[string]resourcet.TestResource)

	var empty resourcet.TestResource
	resources, err := empty.List(ctx)
	require.NoError(t, err)
	assert.Len(t, resources, 2)

	var listOut bytes.Buffer
	listCmd := &ResourceListCmd[resourcet.TestResource]{}
	err = listCmd.Run(ctx, testConfig(&listOut), sandbox)
	require.NoError(t, err)

	output := listOut.String()
	assert.Contains(t, output, "test1")
	assert.Contains(t, output, "test2")
	assert.Contains(t, output, "id-test1")
	assert.Contains(t, output, "id-test2")

	t.Run("quiet", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceListCmd[resourcet.TestResource]{
			FormatOpts: FormatOpts{
				Quiet: true,
			},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)
		assert.Equal(t, "test1\ntest2\n", out.String())

		out.Reset()
		cmd.Name = []string{"test2"}
		err = cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)
		assert.Equal(t, "test2\n", out.String())
	})

	t.Run("raw", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceListCmd[resourcet.TestResource]{
			FormatOpts: FormatOpts{
				Raw: true,
			},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, `"Name": "test1"`)
		assert.Contains(t, output, `"ID": "id-test1"`)

		out.Reset()
		cmd.Name = []string{"test1"}
		err = cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output = out.String()
		assert.Contains(t, output, `"Name": "test1"`)
		assert.Contains(t, output, `"ID": "id-test1"`)
		assert.NotContains(t, output, `"Name": "test2"`)
	})

	t.Run("format", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceListCmd[resourcet.TestResource]{
			FormatOpts: FormatOpts{
				Format: "{{range .}}{{.name}}\n{{end}}",
			},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)
		assert.Equal(t, "test1\ntest2\n", out.String())

		out.Reset()
		cmd.Name = []string{"test2", "test1"}
		err = cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)
		assert.Equal(t, "test2\ntest1\n", out.String())
	})

	t.Run("field", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceListCmd[resourcet.TestResource]{
			FormatOpts: FormatOpts{
				Field: []string{"name", "id"},
			},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "test1")
		assert.Contains(t, output, "id-test1")
		assert.NotContains(t, output, "https://example.com")

		out.Reset()
		cmd.Name = []string{"test1"}
		err = cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output = out.String()
		assert.Contains(t, output, "test1")
		assert.Contains(t, output, "id-test1")
		assert.NotContains(t, output, "test2")
	})

	t.Run("filter", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceListCmd[resourcet.TestResource]{
			FormatOpts: FormatOpts{
				Filter: []string{"name==test1"},
			},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "test1")
		assert.NotContains(t, output, "test2")

		out.Reset()
		cmd.Name = []string{"test1", "test2"}
		err = cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output = out.String()
		assert.Contains(t, output, "test1")
		assert.NotContains(t, output, "test2")
	})
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	sandbox := &resource.Sandbox{}

	cloned, err := copystructure.Copy(baseTestStore)
	require.NoError(t, err)
	resourcet.TestStore = cloned.(map[string]resourcet.TestResource)

	var empty resourcet.TestResource
	resources, err := empty.Get(ctx, []string{"test1"})
	require.NoError(t, err)
	require.Len(t, resources, 1)

	test := resources[0].(resourcet.TestResource)
	assert.Equal(t, "test1", test.Name)
	assert.Equal(t, "id-test1", test.ID)
	assert.Equal(t, 42, test.Settings.X)
	assert.Equal(t, "hello", test.Settings.Y)

	fields, err := test.Fields()
	require.NoError(t, err)
	assert.NotEmpty(t, fields)

	var inspectOut bytes.Buffer
	inspectCmd := &ResourceInspectCmd[resourcet.TestResource]{
		Name: []string{"test1"},
	}
	err = inspectCmd.Run(ctx, testConfig(&inspectOut), sandbox)
	require.NoError(t, err)

	output := inspectOut.String()
	assert.Contains(t, output, "test1")
	assert.Contains(t, output, "id-test1")
	assert.Contains(t, output, "https://example.com")

	t.Run("no_args", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceInspectCmd[resourcet.TestResource]{
			Name: []string{},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.Error(t, err)
	})

	t.Run("multiple", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceInspectCmd[resourcet.TestResource]{
			Name: []string{"test1", "test2"},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "test1")
		assert.Contains(t, output, "test2")
		assert.Contains(t, output, "id-test1")
		assert.Contains(t, output, "id-test2")
	})

	t.Run("quiet", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceInspectCmd[resourcet.TestResource]{
			Name: []string{"test1"},
			FormatOpts: FormatOpts{
				Quiet: true,
			},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)
		assert.Equal(t, "test1\n", out.String())

		out.Reset()
		cmd.Name = []string{"test2", "test1"}
		err = cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)
		assert.Equal(t, "test2\ntest1\n", out.String())
	})

	t.Run("raw", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceInspectCmd[resourcet.TestResource]{
			Name: []string{"test1"},
			FormatOpts: FormatOpts{
				Raw: true,
			},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, `"Name": "test1"`)
		assert.Contains(t, output, `"ID": "id-test1"`)

		out.Reset()
		cmd.Name = []string{"test1", "test2"}
		err = cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output = out.String()
		assert.Contains(t, output, `"Name": "test1"`)
		assert.Contains(t, output, `"ID": "id-test1"`)
		assert.Contains(t, output, `"Name": "test2"`)
		assert.Contains(t, output, `"ID": "id-test2"`)
	})

	t.Run("format", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceInspectCmd[resourcet.TestResource]{
			Name: []string{"test1"},
			FormatOpts: FormatOpts{
				Format: "{{range .}}{{.name}}: {{.url}}\n{{end}}",
			},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)
		assert.Equal(t, "test1: https://example.com\n", out.String())

		out.Reset()
		cmd.Name = []string{"test2", "test1"}
		err = cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)
		assert.Equal(t, "test2: https://example.org\ntest1: https://example.com\n", out.String())
	})

	t.Run("field", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceInspectCmd[resourcet.TestResource]{
			Name: []string{"test1"},
			FormatOpts: FormatOpts{
				Field: []string{"id", "url"},
			},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "id-test1")
		assert.Contains(t, output, "https://example.com")

		out.Reset()
		cmd.Name = []string{"test1", "test2"}
		err = cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output = out.String()
		assert.Contains(t, output, "id-test1")
		assert.Contains(t, output, "https://example.com")
		assert.Contains(t, output, "id-test2")
		assert.Contains(t, output, "https://example.org")
	})

	t.Run("filter", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &ResourceInspectCmd[resourcet.TestResource]{
			Name: []string{"test1"},
			FormatOpts: FormatOpts{
				Filter: []string{"id==id-test1"},
			},
		}
		err := cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output := out.String()
		assert.Contains(t, output, "test1")

		out.Reset()
		cmd.Name = []string{"test1", "test2"}
		err = cmd.Run(ctx, testConfig(&out), sandbox)
		require.NoError(t, err)

		output = out.String()
		assert.Contains(t, output, "test1")
		assert.NotContains(t, output, "test2")
	})
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	resourcet.TestStore = map[string]resourcet.TestResource{}

	var empty resourcet.TestResource
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

	created := res.(resourcet.TestResource)
	assert.Equal(t, "test-new", created.Name)
	assert.Equal(t, 100, created.Settings.X)
	assert.Equal(t, "created", created.Settings.Y)
	assert.Contains(t, resourcet.TestStore, "test-new")
}

func TestEdit(t *testing.T) {
	ctx := context.Background()

	editStore := map[string]resourcet.TestResource{
		"test-edit": {
			ID:   "id-edit",
			Name: "test-edit",
			URL:  "https://example.com",
			Settings: resourcet.TestSettings{
				X: 10,
				Y: "original",
			},
		},
	}
	cloned, err := copystructure.Copy(editStore)
	require.NoError(t, err)
	resourcet.TestStore = cloned.(map[string]resourcet.TestResource)

	var empty resourcet.TestResource
	resources, err := empty.Get(ctx, []string{"test-edit"})
	require.NoError(t, err)
	require.Len(t, resources, 1)

	target := resources[0]
	templateFields, err := target.Fields()
	require.NoError(t, err)

	for key, field := range resource.IterFields(templateFields) {
		if field.Edit == nil {
			continue
		}
		switch key.String() {
		case "settings.x":
			field.Edit.Set = 999
		case "settings.y":
			field.Edit.Set = "modified"
		}
	}

	res, err := empty.Edit(ctx, target, templateFields)
	require.NoError(t, err)

	edited := res.(resourcet.TestResource)
	assert.Equal(t, "test-edit", edited.Name)
	assert.Equal(t, "id-edit", edited.ID)
	assert.Equal(t, 999, edited.Settings.X)
	assert.Equal(t, "modified", edited.Settings.Y)

	stored := resourcet.TestStore["test-edit"]
	assert.Equal(t, 999, stored.Settings.X)
	assert.Equal(t, "modified", stored.Settings.Y)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	deleteStore := map[string]resourcet.TestResource{
		"test-delete": {
			ID:   "id-delete",
			Name: "test-delete",
			URL:  "https://example.com",
		},
		"test-keep": {
			ID:   "id-keep",
			Name: "test-keep",
			URL:  "https://example.org",
		},
	}
	cloned, err := copystructure.Copy(deleteStore)
	require.NoError(t, err)
	resourcet.TestStore = cloned.(map[string]resourcet.TestResource)

	var empty resourcet.TestResource
	resources, err := empty.Get(ctx, []string{"test-delete"})
	require.NoError(t, err)
	require.Len(t, resources, 1)

	err = empty.Delete(ctx, resources)
	require.NoError(t, err)

	assert.NotContains(t, resourcet.TestStore, "test-delete")
	assert.Contains(t, resourcet.TestStore, "test-keep")

	resources, err = empty.Get(ctx, []string{"test-delete"})
	require.NoError(t, err)
	assert.Empty(t, resources)
}

func testConfig(out io.Writer) *config.Config {
	return &config.Config{
		Context: context.Background(),
		Stdin:   &bytes.Buffer{},
		Stdout:  out,
		Stderr:  out,
	}
}
