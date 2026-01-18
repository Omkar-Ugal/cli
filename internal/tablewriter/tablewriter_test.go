// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package tablewriter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicTable(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
| Name | Age | City |
| :--- | :---: | ---: |
| Alice | 30 | New York |
| Bob | 25 | London |
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
| Name  | Age |     City |
| :---- | :-: | -------: |
| Alice | 30  | New York |
| Bob   | 25  |   London |
`
	require.Equal(t, expected, buf.String())
}

func TestLeftAlignedTable(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
| Column1 | Column2 | Column3 |
| :--- | :--- | :--- |
| Value1 | Value2 | Value3 |
| Short | Medium Value | Long Value Here |
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
| Column1 | Column2      | Column3         |
| :------ | :----------- | :-------------- |
| Value1  | Value2       | Value3          |
| Short   | Medium Value | Long Value Here |
`
	require.Equal(t, expected, buf.String())
}

func TestRightAlignedTable(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
| Product | Price | Quantity |
| ---: | ---: | ---: |
| Apple | 1.50 | 10 |
| Banana | 0.75 | 25 |
| Orange | 2.00 | 15 |
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
| Product | Price | Quantity |
| ------: | ----: | -------: |
|   Apple |  1.50 |       10 |
|  Banana |  0.75 |       25 |
|  Orange |  2.00 |       15 |
`
	require.Equal(t, expected, buf.String())
}

func TestCenterAlignedTable(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
| Status | Count | Percentage |
| :---: | :---: | :---: |
| Active | 42 | 65% |
| Inactive | 23 | 35% |
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
|  Status  | Count | Percentage |
| :------: | :---: | :--------: |
|  Active  |  42   |    65%     |
| Inactive |  23   |    35%     |
`
	require.Equal(t, expected, buf.String())
}

func TestNoAlignmentTable(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
| Item | Description | Value |
| --- | --- | --- |
| Item1 | This is item one | 100 |
| Item2 | This is item two | 200 |
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
| Item  | Description      | Value |
| ----- | ---------------- | ----- |
| Item1 | This is item one | 100   |
| Item2 | This is item two | 200   |
`
	require.Equal(t, expected, buf.String())
}

func TestMixedAlignmentTable(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
| Name | Score | Grade | Notes |
| :--- | ---: | :---: | --- |
| Alice | 95 | A | Good |
| Bob | 87 | B | OK |
| Charlie | 92 | A | Great |
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
| Name    | Score | Grade | Notes |
| :------ | ----: | :---: | ----- |
| Alice   |    95 |   A   | Good  |
| Bob     |    87 |   B   | OK    |
| Charlie |    92 |   A   | Great |
`
	require.Equal(t, expected, buf.String())
}

func TestEmptyTable(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	err := w.Flush()
	require.NoError(t, err)
	require.Empty(t, buf.String())
}

func TestIncrementalWrite(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	_, err := w.Write([]byte("| Name | Value |\n"))
	require.NoError(t, err)
	_, err = w.Write([]byte("| :--- | ---: |\n"))
	require.NoError(t, err)
	_, err = w.Write([]byte("| Test | 123 |\n"))
	require.NoError(t, err)

	err = w.Flush()
	require.NoError(t, err)

	expected := `| Name | Value |
| :--- | ----: |
| Test |   123 |
`
	require.Equal(t, expected, buf.String())
}

func TestWhitespaceHandling(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
| Name  |  Age  |  City   |
| :--- | :---: | ---: |
|  Alice  |  30  |  New York  |
|  Bob  |  25  |  London  |
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
| Name  | Age |     City |
| :---- | :-: | -------: |
| Alice | 30  | New York |
| Bob   | 25  |   London |
`
	require.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(buf.String()))
}

func TestMultipleTables(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
| Name | Age |
| :-- | --: |
| Alice | 30 |

| City | Country |
| :--- | :----- |
| London | UK |
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
| Name  | Age |
| :---- | --: |
| Alice |  30 |

| City   | Country |
| :----- | :------ |
| London | UK      |
`
	require.Equal(t, expected, buf.String())
}

func TestInterveningContent(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
| Name | Value |
| :-- | ---: |
| First | 100 |

Some text here

| Another | Data |
| :------ | :--- |
| Second | 200 |
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
| Name  | Value |
| :---- | ----: |
| First |   100 |

Some text here

| Another | Data |
| :------ | :--- |
| Second  | 200  |
`
	require.Equal(t, expected, buf.String())
}

func TestPassthroughLines(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
This is regular text
No pipes here
| Name | Age |
| :--- | --: |
| Alice | 30 |
More regular text
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
This is regular text
No pipes here
| Name  | Age |
| :---- | --: |
| Alice |  30 |
More regular text
`
	require.Equal(t, expected, buf.String())
}

func TestMismatchedColumns(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
| Name | Age | City |
| :--- | :-: | ---: |
| Alice | 30 |
| Bob | 25 | London | Extra |
| Carol |
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
| Name  | Age |   City |
| :---- | :-: | -----: |
| Alice | 30  |        |
| Bob   | 25  | London |
| Carol |     |        |
`
	require.Equal(t, expected, buf.String())
}

func TestMismatchedHeadersAndAlignments(t *testing.T) {
	var buf bytes.Buffer
	w := TableWriter(&buf)

	input := `
| Name | Age | City |
| :--- | :-: |
| Alice | 30 | London |

| Name | Age |
| :--- | :-: | ---: |
| Bob | 25 |
`
	_, err := w.Write([]byte(input))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)

	expected := `
| Name  | Age | City   |
| :---- | :-: |        |
| Alice | 30  | London |

| Name | Age |
| :--- | :-: |
| Bob  | 25  |
`
	require.Equal(t, expected, buf.String())
}
