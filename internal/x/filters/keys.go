// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package filters

import (
	"github.com/containerd/containerd/v2/pkg/filters"
)

// Keys extracts all field paths referenced by a filter.
// This is useful for determining which fields need to be resolved
// before the filter can be evaluated.
func Keys(filter filters.Filter) [][]string {
	var keys [][]string
	collectKeys(filter, &keys)
	return keys
}

func collectKeys(filter filters.Filter, keys *[][]string) {
	switch f := filter.(type) {
	case filters.All:
		for _, sub := range f {
			collectKeys(sub, keys)
		}
	case filters.Any:
		for _, sub := range f {
			collectKeys(sub, keys)
		}
	default:
		// For leaf filters, we need to extract the field path.
		// The filter will call the adaptor with the field path when matching.
		// We use a fake adaptor to capture the field path.
		filter.Match(filters.AdapterFunc(func(fieldpath []string) (string, bool) {
			// Make a copy of the fieldpath to avoid aliasing issues
			pathCopy := make([]string, len(fieldpath))
			copy(pathCopy, fieldpath)
			*keys = append(*keys, pathCopy)
			return "", false
		}))
	}
}
