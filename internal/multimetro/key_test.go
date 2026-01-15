// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package multimetro

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testUUID = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

func TestParseKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Key
	}{
		{
			name:     "metro with uuid",
			input:    "ams#" + testUUID,
			expected: Key{Metro: "ams", UUID: testUUID},
		},
		{
			name:     "metro with explicit uuid",
			input:    "ams#uuid:" + testUUID,
			expected: Key{Metro: "ams", UUID: testUUID},
		},
		{
			name:     "metro with name",
			input:    "ams#api-key",
			expected: Key{Metro: "ams", Name: "api-key"},
		},
		{
			name:     "metro with explicit name",
			input:    "ams#name:api-key",
			expected: Key{Metro: "ams", Name: "api-key"},
		},
		{
			name:     "uuid only",
			input:    testUUID,
			expected: Key{UUID: testUUID},
		},
		{
			name:     "explicit uuid only",
			input:    "uuid:" + testUUID,
			expected: Key{UUID: testUUID},
		},
		{
			name:     "name only",
			input:    "api-key",
			expected: Key{Name: "api-key"},
		},
		{
			name:     "explicit name only",
			input:    "name:api-key",
			expected: Key{Name: "api-key"},
		},
		{
			name:     "name is uuid",
			input:    "name:" + testUUID,
			expected: Key{Name: testUUID},
		},
		{
			name:     "explicit uuid non-uuid",
			input:    "uuid:not-a-uuid",
			expected: Key{UUID: "not-a-uuid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseKey(tt.input))
		})
	}
}

func TestKeyString(t *testing.T) {
	tests := []struct {
		name     string
		key      Key
		expected string
	}{
		{
			name:     "uuid output",
			key:      Key{UUID: testUUID},
			expected: testUUID,
		},
		{
			name:     "uuid output with metro",
			key:      Key{Metro: "ams", UUID: testUUID},
			expected: "ams#" + testUUID,
		},
		{
			name:     "uuid needs prefix for non-uuid",
			key:      Key{UUID: "not-a-uuid"},
			expected: "uuid:not-a-uuid",
		},
		{
			name:     "uuid needs prefix for uuid prefix",
			key:      Key{UUID: "uuid:foo"},
			expected: "uuid:uuid:foo",
		},
		{
			name:     "name output",
			key:      Key{Name: "api-key"},
			expected: "api-key",
		},
		{
			name:     "name output with metro",
			key:      Key{Metro: "ams", Name: "api-key"},
			expected: "ams#api-key",
		},
		{
			name:     "name needs prefix for uuid",
			key:      Key{Name: testUUID},
			expected: "name:" + testUUID,
		},
		{
			name:     "name needs prefix for name prefix",
			key:      Key{Name: "name:foo"},
			expected: "name:name:foo",
		},
		{
			name:     "name needs prefix for uuid prefix",
			key:      Key{Name: "uuid:foo"},
			expected: "name:uuid:foo",
		},
		{
			name:     "metro name needs prefix",
			key:      Key{Metro: "ams", Name: testUUID},
			expected: "ams#name:" + testUUID,
		},
		{
			name:     "metro uuid non-uuid",
			key:      Key{Metro: "ams", UUID: "not-a-uuid"},
			expected: "ams#uuid:not-a-uuid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.key.String())
		})
	}
}

func TestParseKeys(t *testing.T) {
	inputs := []string{
		"ams#" + testUUID,
		"lhr#name:api-key",
		"uuid:not-a-uuid",
	}
	expected := Keys{
		{Metro: "ams", UUID: testUUID},
		{Metro: "lhr", Name: "api-key"},
		{UUID: "not-a-uuid"},
	}

	assert.Equal(t, expected, ParseKeys(inputs))
}
