// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package tableutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReducePadding(t *testing.T) {
	widths := []int{3, 1, 2}
	reduced := ReducePadding(widths, 3)
	require.Equal(t, 3, reduced)
	require.Equal(t, []int{2, 0, 1}, widths)
}

func TestReducePaddingCannotGoBelowZero(t *testing.T) {
	widths := []int{1, 1}
	reduced := ReducePadding(widths, 5)
	require.Equal(t, 2, reduced)
	require.Equal(t, []int{0, 0}, widths)
}

func TestReducePaddingNoopOnZeros(t *testing.T) {
	widths := []int{0, 0}
	reduced := ReducePadding(widths, 5)
	require.Equal(t, 0, reduced)
	require.Equal(t, []int{0, 0}, widths)
}

func TestGrowPaddingSkipsLastWhenMultiple(t *testing.T) {
	widths := []int{0, 0, 0}
	grown := GrowPadding(widths, 4)
	require.Equal(t, 4, grown)
	require.Equal(t, []int{2, 2, 0}, widths)
}

func TestGrowPaddingSingleWidth(t *testing.T) {
	widths := []int{0}
	grown := GrowPadding(widths, 3)
	require.Equal(t, 3, grown)
	require.Equal(t, []int{3}, widths)
}

func TestReduceColumns(t *testing.T) {
	widths := []int{3, 1, 3}
	ReduceColumns(widths, 2)
	require.Equal(t, []int{2, 1, 2}, widths)
}
