// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package multimetro

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"unikraft.com/x/filters"
)

func TestFilterMetros(t *testing.T) {
	metros := []string{"metro1", "metro2", "metro3"}
	tests := []struct {
		name   string
		filter []string
		input  []string
		output []string
	}{
		{
			name:   "no filter",
			filter: []string{},
			input:  metros,
			output: []string{"metro1", "metro2", "metro3"},
		},
		{
			name:   "single metro equals",
			filter: []string{"metro==metro1"},
			input:  metros,
			output: []string{"metro1"},
		},
		{
			name:   "multiple metro equals",
			filter: []string{"metro==metro2", "metro==metro3"},
			input:  metros,
			output: []string{"metro2", "metro3"},
		},
		{
			name:   "comma separated metro equals", // contradictory
			filter: []string{"metro==metro2,metro==metro3"},
			input:  metros,
			output: []string{},
		},
		{
			name:   "single metro not equals",
			filter: []string{"metro!=metro1"},
			input:  metros,
			output: []string{"metro2", "metro3"},
		},
		{
			name:   "contradictory expressions metros",
			filter: []string{"metro!=metro1,metro==metro1"},
			input:  metros,
			output: []string{},
		},
		{
			name:   "other fields",
			filter: []string{"otherfield==value"},
			input:  metros,
			output: []string{"metro1", "metro2", "metro3"},
		},
		{
			name:   "other fields with metro",
			filter: []string{"metro==metro2", "otherfield==value"},
			input:  metros,
			output: []string{"metro1", "metro2", "metro3"},
		},
		{
			name:   "comma separated other fields with metro",
			filter: []string{"metro==metro2,otherfield==value"},
			input:  metros,
			output: []string{"metro2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := filters.ParseAll(tt.filter...)
			require.NoError(t, err)

			result := filterMetros(tt.input, filter)
			assert.Equal(t, tt.output, result)
		})
	}
}
