// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package kvwriter

import (
	"bytes"
	"fmt"
	"io"
	"slices"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lunixbochs/vtclean"
	"github.com/muesli/termenv"
)

type keyValueWriter struct {
	w      io.Writer
	buffer []byte

	splits []string
	cut    int

	cells []cell
}

type cell struct {
	key      []byte
	cleankey []byte
	value    []byte
	split    []byte
}

func (entry cell) width() int {
	return len(entry.cleankey) + len(entry.split)
}

// KeyValueWriter returns a Writer that formats key-value pairs written to it.
// Each line written to the Writer should be in the format "key: value".
//
// The keys will be aligned based on the longest key, and styling rules are
// applied to them based on their leading whitespace after the specified indent
// is removed.
func KeyValueWriter(w io.Writer, indent string, splits ...string) WriteFlusher {
	if len(splits) == 0 {
		splits = []string{": "}
	}
	return &keyValueWriter{w: w, cut: len(indent), splits: splits}
}

func (b *keyValueWriter) Write(p []byte) (n int, err error) {
	b.buffer = append(b.buffer, p...)

	lastNewline := bytes.LastIndexByte(b.buffer, '\n')
	if lastNewline == -1 {
		return len(p), nil
	}

	lines := bytes.Lines(b.buffer[:lastNewline+1])
	b.buffer = b.buffer[lastNewline+1:]

	for line := range lines {
		b.cells = append(b.cells, b.parseLine(line))
	}
	return len(p), nil
}

func (b *keyValueWriter) parseLine(line []byte) cell {
	parsed, ok := b.splitLine(line)
	if ok {
		return parsed
	}
	return cell{value: line}
}

func (b *keyValueWriter) splitLine(line []byte) (cell, bool) {
	var best cell
	bestIndex := -1

	for _, split := range b.splits {
		if split == "" {
			continue
		}
		splitBytes := []byte(split)
		idx := bytes.Index(line, splitBytes)
		if idx == -1 {
			continue
		}
		value := line[idx+len(splitBytes):]
		if bestIndex != -1 && idx > bestIndex {
			continue
		}
		if bestIndex != -1 && idx == bestIndex && len(splitBytes) <= len(best.split) {
			continue
		}
		key := line[:idx]
		best = cell{
			key:      key,
			cleankey: []byte(vtclean.Clean(string(key), false)),
			value:    bytes.TrimSpace(value),
			split:    splitBytes,
		}
		bestIndex = idx
	}
	if bestIndex == -1 {
		return cell{}, false
	}
	return best, true
}

func (b *keyValueWriter) flush() error {
	if len(b.buffer) > 0 {
		_, err := b.Write([]byte{'\n'})
		if err != nil {
			return err
		}
	}

	color := lipgloss.ColorProfile() != termenv.Ascii

	maxKeyLen := 0
	for _, entry := range b.cells {
		if entry.key == nil {
			continue
		}
		maxKeyLen = max(maxKeyLen, entry.width())
	}

	for _, entry := range b.cells {
		if entry.key == nil {
			if _, err := fmt.Fprint(b.w, string(entry.value)); err != nil {
				return err
			}
			continue
		}
		cleankey := entry.cleankey
		if len(cleankey) > 0 {
			var styleSeq, resetSeq ansi.Style
			if color {
				// NOTE: would be nice to use lipgloss styles here, but lipgloss
				// makes liberal use of full reset codes which can mess up if the
				// target text contains it's own styles.
				if b.cut < len(cleankey) && slices.Contains([]byte(" \t-"), cleankey[b.cut]) {
					// italic
					styleSeq = ansi.NewStyle(ansi.AttrItalic)
					resetSeq = ansi.NewStyle(ansi.AttrNoItalic)
				} else {
					// bold
					styleSeq = ansi.NewStyle(ansi.AttrBold)
					resetSeq = ansi.NewStyle(ansi.AttrNormalIntensity)
				}
			}

			padding := max(0, maxKeyLen-entry.width())
			line := []byte{}
			if styleSeq != nil {
				line = append(line, []byte(styleSeq.String())...)
			}
			line = append(line, entry.key...)
			if styleSeq != nil {
				line = append(line, []byte(resetSeq.String())...)
			}
			line = append(line, entry.split...)
			if len(entry.value) > 0 {
				if padding > 0 {
					line = append(line, bytes.Repeat([]byte(" "), padding)...)
				}
				line = append(line, entry.value...)
			}
			line = append(line, '\n')
			if _, err := b.w.Write(line); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *keyValueWriter) Flush() error {
	if err := b.flush(); err != nil {
		return err
	}
	if len(b.buffer) > 0 {
		if _, err := b.w.Write(b.buffer); err != nil {
			return err
		}
		b.buffer = nil
	}
	if tw, ok := b.w.(Flusher); ok {
		return tw.Flush()
	}
	return nil
}
