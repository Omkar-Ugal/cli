// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package patch

import "unikraft.com/cli/internal/resource"

func FilterPatchableFields(fields []resource.Field) []resource.Field {
	return resource.FilterFields(fields, func(field resource.Field) resource.FilterResult {
		if field.Patch != nil {
			return resource.FilterInclude
		}
		return resource.FilterPrune
	})
}

func FilterCreatableFields(fields []resource.Field) []resource.Field {
	return resource.FilterFields(fields, func(field resource.Field) resource.FilterResult {
		if field.Create != nil {
			return resource.FilterInclude
		}
		return resource.FilterPrune
	})
}
