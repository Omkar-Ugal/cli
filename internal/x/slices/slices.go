// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package slices

// Flatten flattens a slice of slices into a single slice.
func Flatten[T any](slices [][]T) []T {
	total := 0
	for _, slice := range slices {
		total += len(slice)
	}
	result := make([]T, 0, total)
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}
