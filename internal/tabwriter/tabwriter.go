// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package tabwriter

import (
	"bytes"
	"io"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"unikraft.com/x/guesstermwidth"
)

type tabWriter struct {
	w      io.Writer
	buffer []byte
	rows   []row

	mincolwidth int
	padding     int

	maxwidth int
}

type row struct {
	cells []cell
}

type cell struct {
	raw   []byte
	width int
}

type TabWriterOpt func(*tabWriter)

func WithMinColumnWidth(width int) TabWriterOpt {
	return func(t *tabWriter) {
		t.mincolwidth = max(0, width)
	}
}

func WithMaxWidth(width int) TabWriterOpt {
	return func(t *tabWriter) {
		t.maxwidth = max(0, width)
	}
}

func WithMaxScreenWidth() TabWriterOpt {
	return func(t *tabWriter) {
		if !isTTY(t.w) {
			return
		}
		t.maxwidth = max(0, guesstermwidth.GuessTermWidth(t.w))
	}
}

func WithPadding(padding int) TabWriterOpt {
	return func(t *tabWriter) {
		t.padding = max(0, padding)
	}
}

// TabWriter returns a Writer that formats tab-aligned columns.
// Input should be formatted using tab characters between columns.
func TabWriter(w io.Writer, opts ...TabWriterOpt) WriteFlusher {
	tw := &tabWriter{
		w:       w,
		padding: 2,
	}
	for _, opt := range opts {
		opt(tw)
	}
	return tw
}

func (t *tabWriter) Write(p []byte) (n int, err error) {
	t.buffer = append(t.buffer, p...)
	lastNewline := bytes.LastIndexByte(t.buffer, '\n')
	if lastNewline == -1 {
		return len(p), nil
	}

	lines := bytes.Lines(t.buffer[:lastNewline+1])
	t.buffer = t.buffer[lastNewline+1:]

	for line := range lines {
		line = bytes.TrimSuffix(line, []byte("\n"))
		if bytes.IndexByte(line, '\t') != -1 {
			t.rows = append(t.rows, t.parseRow(line))
			continue
		}
		if err := t.flushRows(); err != nil {
			return 0, err
		}
		if _, err := t.w.Write(append(line, '\n')); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

func (t *tabWriter) parseRow(line []byte) row {
	parts := bytes.Split(line, []byte("\t"))
	cells := make([]cell, len(parts))
	for i, part := range parts {
		cells[i] = cell{raw: part, width: ansi.StringWidth(string(part))}
	}

	return row{cells: cells}
}

func (t *tabWriter) flushRows() error {
	if len(t.rows) == 0 {
		return nil
	}

	colCount := 0
	for _, row := range t.rows {
		colCount = max(colCount, len(row.cells))
	}

	colContent := make([]int, colCount)
	for _, row := range t.rows {
		for idx, cell := range row.cells {
			colContent[idx] = max(colContent[idx], cell.width)
		}
	}

	colPadding := make([]int, colCount)
	for idx, content := range colContent {
		colWidth := max(content+t.padding, t.mincolwidth)
		colPadding[idx] = max(0, colWidth-content)
	}
	colPadding[len(colPadding)-1] = 0

	if t.maxwidth > 0 {
		widthSum := 0
		for _, p := range colPadding {
			widthSum += p
		}
		for _, w := range colContent {
			widthSum += w
		}

		over := widthSum - t.maxwidth
		over -= reducePadding(colPadding, over)
		if over > 0 {
			reduceColumns(colContent, over)
		}
	}

	for _, row := range t.rows {
		var line []byte
		for i, cell := range row.cells {
			contentWidth := colContent[i]
			padWidth := colPadding[i]
			raw := string(cell.raw)
			if cell.width > contentWidth {
				raw = ansi.Truncate(raw, max(0, contentWidth), "…")
			}
			trimmedWidth := ansi.StringWidth(raw)
			contentPad := max(0, contentWidth-trimmedWidth)
			line = append(line, []byte(raw)...)
			if i < len(row.cells)-1 {
				line = append(line, bytes.Repeat([]byte(" "), contentPad+padWidth)...)
			}
		}
		line = append(line, '\n')
		if _, err := t.w.Write(line); err != nil {
			return err
		}
	}

	t.rows = nil
	return nil
}

func reducePadding(widths []int, reduce int) int {
	if reduce <= 0 {
		return 0
	}
	const minPadding = 1
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

func reduceColumns(widths []int, reduce int) {
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

func (t *tabWriter) Flush() error {
	if len(t.buffer) > 0 {
		if _, err := t.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	if err := t.flushRows(); err != nil {
		return err
	}
	if tw, ok := t.w.(Flusher); ok {
		return tw.Flush()
	}
	return nil
}

func isTTY(out io.Writer) bool {
	fdWriter, ok := out.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	return term.IsTerminal(fdWriter.Fd())
}
