// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package tableutil

// ReducePadding reduces padding widths down to zero.
// It returns the actual amount reduced.
func ReducePadding(widths []int, reduce int) int {
	if reduce <= 0 {
		return 0
	}
	const minPadding = 0
	remaining := reduce
	for remaining > 0 {
		changed := false
		for i := range widths {
			if widths[i] <= minPadding {
				continue
			}
			widths[i]--
			remaining--
			changed = true
			if remaining == 0 {
				break
			}
		}
		if !changed {
			break
		}
	}
	return reduce - remaining
}

// ReduceColumns reduces the widest columns first.
func ReduceColumns(widths []int, reduce int) {
	if reduce <= 0 {
		return
	}
	remaining := reduce
	for remaining > 0 {
		maxIdx := -1
		maxWidth := 0
		for i, width := range widths {
			if width > maxWidth {
				maxWidth = width
				maxIdx = i
			}
		}
		if maxIdx == -1 {
			break
		}
		widths[maxIdx]--
		remaining--
	}
}

// GrowPadding distributes extra width across padding columns.
// It returns the actual amount grown.
func GrowPadding(widths []int, grow int) int {
	if grow <= 0 || len(widths) == 0 {
		return 0
	}
	remaining := grow
	last := len(widths) - 1
	for remaining > 0 {
		changed := false
		for i := range widths {
			if len(widths) > 1 && i == last {
				continue
			}
			widths[i]++
			remaining--
			changed = true
			if remaining == 0 {
				break
			}
		}
		if !changed {
			break
		}
	}
	return grow - remaining
}
