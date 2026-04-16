// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package patch

import (
	"reflect"

	"unikraft.com/cli/internal/resource"
)

// FilterEditFields filters to fields that have a pending edit patch.
func FilterEditFields(fields []resource.Field) []resource.Field {
	return resource.FilterFields(fields, func(field resource.Field) resource.FilterResult {
		if field.Edit == nil {
			return resource.FilterPrune
		}
		if field.Edit.Set == nil && field.Edit.Add == nil && field.Edit.Del == nil {
			return resource.FilterPrune
		}
		return resource.FilterInclude
	})
}

// FilterCreateFields filters to fields that have a pending create patch.
func FilterCreateFields(fields []resource.Field) []resource.Field {
	return resource.FilterFields(fields, func(field resource.Field) resource.FilterResult {
		if field.Create == nil {
			return resource.FilterPrune
		}
		if field.Create.Set == nil && field.Create.Add == nil && field.Create.Del == nil {
			return resource.FilterPrune
		}
		return resource.FilterInclude
	})
}

// FilterDisplayableEditFields filters fields for visual editing display.
// Includes fields that have an Edit patch and a non-nil Value to show.
func FilterDisplayableEditFields(fields []resource.Field) []resource.Field {
	return resource.FilterFields(fields, func(field resource.Field) resource.FilterResult {
		if field.Edit == nil {
			return resource.FilterPrune
		}
		if reflect.ValueOf(field.Edit.Set).IsZero() {
			return resource.FilterPrune
		}
		return resource.FilterInclude
	})
}

// FilterDisplayableCreateFields filters fields for visual creation display.
// Includes all fields that have a Create patch (even with zero values, so users can fill them in).
func FilterDisplayableCreateFields(fields []resource.Field) []resource.Field {
	return resource.FilterFields(fields, func(field resource.Field) resource.FilterResult {
		if field.Create == nil {
			return resource.FilterPrune
		}
		if !field.Create.Required {
			return resource.FilterPrune
		}
		return resource.FilterInclude
	})
}
