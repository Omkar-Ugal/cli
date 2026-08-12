// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEscapeMdx(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text is untouched",
			in:   "no special characters here",
			want: "no special characters here",
		},
		{
			name: "angle brackets are escaped",
			in:   "Forward local port to <INSTANCE>:DEST_PORT.",
			want: `Forward local port to \<INSTANCE>:DEST_PORT.`,
		},
		{
			name: "curly braces are escaped",
			in:   `Produces: {"user":{"name":"Alice"}}`,
			want: `Produces: \{"user":\{"name":"Alice"\}\}`,
		},
		{
			name: "inline code spans are left verbatim",
			in:   "Use `<endpoint>` or `{key: value}` on the command line.",
			want: "Use `<endpoint>` or `{key: value}` on the command line.",
		},
		{
			name: "mixed prose and inline code on one line",
			in:   "A <bad> tag next to `<ok>` code.",
			want: `A \<bad> tag next to ` + "`<ok>`" + " code.",
		},
		{
			name: "fenced code blocks are left verbatim",
			in:   "before\n```\n<raw> {raw}\n```\nafter <bad> {bad}",
			want: "before\n```\n<raw> {raw}\n```\nafter \\<bad> \\{bad\\}",
		},
		{
			name: "tilde fences are also respected",
			in:   "~~~\n<raw>\n~~~\n<bad>",
			want: "~~~\n<raw>\n~~~\n\\<bad>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, escapeMdx(tc.in))
		})
	}
}
