// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package patch

import (
	"unikraft.com/cli/internal/resource"
)

// MergePatches merges patch values from src into dest fields.
// For each field in dest, if src contains a patch at the same path,
// the src patch value overrides the dest patch value.
// Fields in src that don't exist in dest are appended.
func MergePatches(dest, src []resource.Field) []resource.Field {
	dest = resource.CloneFields(dest)

	// Build maps of src patches by path
	srcEditPatches := make(map[string]*resource.Patch)
	srcCreatePatches := make(map[string]*resource.Patch)
	for key, field := range resource.IterFields(src) {
		keyStr := key.String()
		if field.Edit != nil {
			srcEditPatches[keyStr] = field.Edit
		}
		if field.Create != nil {
			srcCreatePatches[keyStr] = field.Create
		}
	}

	// Track which src patches we've applied
	appliedPatches := make(map[string]bool)

	// Merge src patches into dest fields
	for key, field := range resource.IterFields(dest) {
		keyStr := key.String()

		if srcPatch, ok := srcEditPatches[keyStr]; ok {
			appliedPatches[keyStr] = true
			if field.Edit != nil {
				mergePatch(field.Edit, srcPatch)
			}
		}
		if srcPatch, ok := srcCreatePatches[keyStr]; ok {
			appliedPatches[keyStr] = true
			if field.Create != nil {
				mergePatch(field.Create, srcPatch)
			}
		}
	}

	// Append src fields that weren't in dest
	for key, field := range resource.IterFields(src) {
		keyStr := key.String()
		if appliedPatches[keyStr] {
			continue
		}
		// Only append if there's a Set value
		hasSet := (field.Edit != nil && field.Edit.Set != nil) ||
			(field.Create != nil && field.Create.Set != nil)
		if !hasSet {
			continue
		}
		dest = append(dest, resource.Field{
			Name:   field.Name,
			Edit:   field.Edit,
			Create: field.Create,
		})
	}

	return dest
}

// mergePatch merges src patch values into dest patch.
func mergePatch(dest, src *resource.Patch) {
	if src.Set != nil {
		dest.Set = src.Set
	}
	if src.Add != nil {
		dest.Add = src.Add
	}
	if src.Del != nil {
		dest.Del = src.Del
	}
}
