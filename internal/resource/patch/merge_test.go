// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package patch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"unikraft.com/cli/internal/resource"
)

// patchPaths keys every Set patch by its path, so a field inserted at the
// wrong depth shows up as a differing key.
func patchPaths(fields []resource.Field) map[string]any {
	got := map[string]any{}
	for path, field := range resource.IterFields(fields) {
		if field.Edit != nil && field.Edit.Set != nil {
			got[path.String()] = field.Edit.Set
		}
		if field.Create != nil && field.Create.Set != nil {
			got[path.String()] = field.Create.Set
		}
	}
	return got
}

func TestMergePatches_NestedPathsPreserved(t *testing.T) {
	nested := func(path ...string) []resource.Field {
		var set any = "value"
		field := resource.Field{Name: path[len(path)-1], Edit: &resource.Patch{Set: set}}
		for i := len(path) - 2; i >= 0; i-- {
			field = resource.Field{Name: path[i], Subfields: []resource.Field{field}}
		}
		return []resource.Field{field}
	}

	t.Run("absent parent is created", func(t *testing.T) {
		got := MergePatches(nil, nested("service", "domains"))
		assert.Equal(t, map[string]any{"service.domains": "value"}, patchPaths(got))
	})

	t.Run("existing parent is reused", func(t *testing.T) {
		dest := []resource.Field{{
			Name:   "service",
			Create: &resource.Patch{Set: "my-group"},
		}}
		got := MergePatches(dest, nested("service", "domains"))
		assert.Equal(t, map[string]any{
			"service":         "my-group",
			"service.domains": "value",
		}, patchPaths(got))

		require.Len(t, got, 1, "descendant must nest, not become a sibling")
		require.Len(t, got[0].Subfields, 1)
		assert.Equal(t, "domains", got[0].Subfields[0].Name)
	})

	t.Run("siblings share one parent", func(t *testing.T) {
		src := append(nested("service", "domains"), nested("service", "services")...)
		got := MergePatches(nil, src)
		assert.Equal(t, map[string]any{
			"service.domains":  "value",
			"service.services": "value",
		}, patchPaths(got))
		require.Len(t, got, 1)
		assert.Len(t, got[0].Subfields, 2)
	})

	t.Run("deep path", func(t *testing.T) {
		got := MergePatches(nil, nested("a", "b", "c"))
		assert.Equal(t, map[string]any{"a.b.c": "value"}, patchPaths(got))
	})

	t.Run("matching path merges rather than inserts", func(t *testing.T) {
		dest := []resource.Field{{
			Name:      "service",
			Subfields: []resource.Field{{Name: "domains", Edit: &resource.Patch{Set: "old"}}},
		}}
		got := MergePatches(dest, nested("service", "domains"))
		assert.Equal(t, map[string]any{"service.domains": "value"}, patchPaths(got))
		require.Len(t, got, 1)
		assert.Len(t, got[0].Subfields, 1)
	})

	t.Run("src is not mutated", func(t *testing.T) {
		src := nested("service", "domains")
		MergePatches([]resource.Field{{Name: "service"}}, src)
		require.Len(t, src, 1)
		assert.Len(t, src[0].Subfields, 1)
	})
}
