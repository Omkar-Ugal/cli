// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

// Package prettydiff provides functionality to render diffs from the
// github.com/sergi/go-diff package in a pretty, colored format.
package prettydiff

import (
	"bytes"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// Render takes a slice of diffs and returns a pretty-printed string
// representation with color coding for additions and deletions.
//
// The diffs are broken down into a line-by-line format, grouped into hunks,
// and rendered with appropriate ANSI color codes for the terminal display.
func Render(diffs []diffmatchpatch.Diff) string {
	var lines []line
	var current []diffmatchpatch.Diff

	for _, diff := range diffs {
		text := diff.Text

		for len(text) > 0 {
			idx := strings.IndexByte(text, '\n')

			if idx == -1 {
				current = append(current, diffmatchpatch.Diff{
					Type: diff.Type,
					Text: text,
				})
				break
			}
			current = append(current, diffmatchpatch.Diff{
				Type: diff.Type,
				Text: text[:idx], // cut the newline character
			})

			lines = append(lines, parseLine(current)...)

			current = nil
			text = text[idx+1:]
		}
	}
	if len(current) > 0 {
		lines = append(lines, parseLine(current)...)
	}

	color := lipgloss.ColorProfile() != termenv.Ascii

	var buff bytes.Buffer
	lines = groupLines(lines)
	for _, line := range lines {
		line.render(&buff, color)
	}
	return buff.String()
}

// groupLines groups lines into hunks (so we group deletions next to deletions,
// instead of alternating between deletion + additions)
func groupLines(lines []line) []line {
	var done []line

	var added []line
	var removed []line

	for _, l := range lines {
		switch l.op {
		case diffmatchpatch.DiffInsert:
			added = append(added, l)
		case diffmatchpatch.DiffDelete:
			removed = append(removed, l)
		case diffmatchpatch.DiffEqual:
			done = append(done, removed...)
			removed = nil
			done = append(done, added...)
			added = nil
			done = append(done, l)
		}
	}
	done = append(done, removed...)
	done = append(done, added...)
	return done
}

type line struct {
	op    diffmatchpatch.Operation
	diffs []diffmatchpatch.Diff
}

func parseLine(lineDiffs []diffmatchpatch.Diff) []line {
	hasInsert, hasDelete := getLineType(lineDiffs)
	if !hasInsert && !hasDelete {
		return []line{{
			op:    diffmatchpatch.DiffEqual,
			diffs: lineDiffs,
		}}
	}

	var lines []line

	// deletions
	if hasInsert && !hasDelete {
		var equalDiffs []diffmatchpatch.Diff
		for _, d := range lineDiffs {
			if d.Type == diffmatchpatch.DiffEqual {
				equalDiffs = append(equalDiffs, d)
			}
		}
		if len(equalDiffs) > 0 {
			lines = append(lines, line{
				op:    diffmatchpatch.DiffDelete,
				diffs: equalDiffs,
			})
		}
	}
	if hasDelete {
		var delDiffs []diffmatchpatch.Diff
		for _, d := range lineDiffs {
			if d.Type == diffmatchpatch.DiffDelete || d.Type == diffmatchpatch.DiffEqual {
				delDiffs = append(delDiffs, d)
			}
		}
		lines = append(lines, line{
			op:    diffmatchpatch.DiffDelete,
			diffs: delDiffs,
		})
	}

	// additions
	if hasDelete && !hasInsert {
		var equalDiffs []diffmatchpatch.Diff
		for _, d := range lineDiffs {
			if d.Type == diffmatchpatch.DiffEqual {
				equalDiffs = append(equalDiffs, d)
			}
		}
		if len(equalDiffs) > 0 {
			lines = append(lines, line{
				op:    diffmatchpatch.DiffInsert,
				diffs: equalDiffs,
			})
		}
	}
	if hasInsert {
		var insDiffs []diffmatchpatch.Diff
		for _, d := range lineDiffs {
			if d.Type == diffmatchpatch.DiffInsert || d.Type == diffmatchpatch.DiffEqual {
				insDiffs = append(insDiffs, d)
			}
		}
		lines = append(lines, line{
			op:    diffmatchpatch.DiffInsert,
			diffs: insDiffs,
		})
	}

	return lines
}

func getLineType(lineDiffs []diffmatchpatch.Diff) (hasInsert bool, hasDelete bool) {
	for _, d := range lineDiffs {
		switch d.Type {
		case diffmatchpatch.DiffInsert:
			hasInsert = true
		case diffmatchpatch.DiffDelete:
			hasDelete = true
		}
	}
	return hasInsert, hasDelete
}

// ANSI color codes for backgrounds
// FIXME: replace with unikraft.com/x/colors
var (
	resetStyle      = ansi.NewStyle(ansi.DefaultBackgroundColorAttr)
	lineRemoveStyle = ansi.NewStyle(ansi.RedBackgroundColorAttr)
	lineAddStyle    = ansi.NewStyle(ansi.GreenBackgroundColorAttr)
	wordRemoveStyle = ansi.NewStyle(ansi.BrightRedBackgroundColorAttr)
	wordAddStyle    = ansi.NewStyle(ansi.BrightGreenBackgroundColorAttr)
)

func (line line) render(buff *bytes.Buffer, color bool) {
	switch line.op {
	case diffmatchpatch.DiffEqual:
		buff.WriteString("  ")
		for _, d := range line.diffs {
			buff.WriteString(d.Text)
		}
		buff.WriteString("\n")
	case diffmatchpatch.DiffDelete:
		if color {
			buff.WriteString(lineRemoveStyle.String())
		}
		buff.WriteString("- ")
		for _, d := range line.diffs {
			if color && d.Type == diffmatchpatch.DiffDelete {
				buff.WriteString(wordRemoveStyle.String())
				buff.WriteString(d.Text)
				buff.WriteString(lineRemoveStyle.String())
			} else {
				buff.WriteString(d.Text)
			}
		}
		if color {
			// EraseLineRight means the whole line (till the right edge of the
			// terminal) is colored but also this then looks super weird if/when the
			// terminal is resized later - leaving out for now
			// buff.WriteString(ansi.EraseLineRight)
			buff.WriteString(resetStyle.String())
		}
		buff.WriteString("\n")
	case diffmatchpatch.DiffInsert:
		if color {
			buff.WriteString(lineAddStyle.String())
		}
		buff.WriteString("+ ")
		for _, d := range line.diffs {
			if color && d.Type == diffmatchpatch.DiffInsert {
				buff.WriteString(wordAddStyle.String())
				buff.WriteString(d.Text)
				buff.WriteString(lineAddStyle.String())
			} else {
				buff.WriteString(d.Text)
			}
		}
		if color {
			// buff.WriteString(ansi.EraseLineRight)
			buff.WriteString(resetStyle.String())
		}
		buff.WriteString("\n")
	}
}
