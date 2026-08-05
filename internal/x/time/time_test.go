// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"7d", 7 * 24 * time.Hour},
		{"2w", 2 * 7 * 24 * time.Hour},
		{"1y", 365 * 24 * time.Hour},
		{"2w1d", 2*7*24*time.Hour + 24*time.Hour},
		{"3d1h", 3*24*time.Hour + time.Hour},
		{"1.5d", 36 * time.Hour},
		{"30m", 30 * time.Minute},
		{"1h30m", time.Hour + 30*time.Minute},
		{"-7d", -7 * 24 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := ParseDuration(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, d)
		})
	}

	t.Run("invalid", func(t *testing.T) {
		_, err := ParseDuration("notaduration")
		assert.Error(t, err)
	})
}
