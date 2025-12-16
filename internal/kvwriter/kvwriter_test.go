// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package kvwriter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lunixbochs/vtclean"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	var buf bytes.Buffer
	w := KeyValueWriter(&buf, "")

	input := `
Name: ExampleApp
Version: 1.0.0
Description: This is an example application.
	`
	_, err := w.Write([]byte(strings.TrimSpace(input)))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
Name:        ExampleApp
Version:     1.0.0
Description: This is an example application.
	`
	clean := vtclean.Clean(buf.String(), false)
	require.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(clean))
}

func TestIndentedKeys(t *testing.T) {
	var buf bytes.Buffer
	w := KeyValueWriter(&buf, "  ")

	input := `
> Name: ExampleApp
> Version: 1.0.0
> Description: This is an example application.
	`
	_, err := w.Write([]byte(strings.TrimSpace(input)))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
> Name:        ExampleApp
> Version:     1.0.0
> Description: This is an example application.
	`
	clean := vtclean.Clean(buf.String(), false)
	require.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(clean))
}

func TestMixedLines(t *testing.T) {
	var buf bytes.Buffer
	w := KeyValueWriter(&buf, "")

	input := `
Name: ExampleApp
This line has no key-value format.
Version: 1.0.0
Another plain line.
Description: This is an example application.
	`
	_, err := w.Write([]byte(strings.TrimSpace(input)))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
Name:        ExampleApp
This line has no key-value format.
Version:     1.0.0
Another plain line.
Description: This is an example application.
	`
	clean := vtclean.Clean(buf.String(), false)
	require.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(clean))
}

func TestMixedLinesIndent(t *testing.T) {
	var buf bytes.Buffer
	w := KeyValueWriter(&buf, "    ")

	input := `
>>>> Name: ExampleApp
Version: 1.0.0
>>>> Description: This is an example application.
	`
	_, err := w.Write([]byte(strings.TrimSpace(input)))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
>>>> Name:        ExampleApp
Version:          1.0.0
>>>> Description: This is an example application.
	`
	clean := vtclean.Clean(buf.String(), false)
	require.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(clean))
}
