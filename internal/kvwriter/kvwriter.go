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

	cut int

	keys      [][]byte
	cleankeys [][]byte
	values    [][]byte
}

// KeyValueWriter returns a Writer that formats key-value pairs written to it.
// Each line written to the Writer should be in the format "key: value".
//
// The keys will be aligned based on the longest key, and styling rules are
// applied to them based on their leading whitespace after the specified indent
// is removed.
func KeyValueWriter(w io.Writer, indent string) WriteFlusher {
	return &keyValueWriter{w: w, cut: len(indent)}
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
		key, value, ok := bytes.Cut(line, []byte(":"))
		if ok {
			b.keys = append(b.keys, key)
			b.cleankeys = append(b.cleankeys, []byte(vtclean.Clean(string(key), false)))
			b.values = append(b.values, bytes.TrimSpace(value))
		} else {
			b.keys = append(b.keys, nil)
			b.cleankeys = append(b.cleankeys, nil)
			b.values = append(b.values, line)
		}
	}
	return len(p), nil
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
	for _, key := range b.cleankeys {
		maxKeyLen = max(maxKeyLen, len(key))
	}

	for i, key := range b.keys {
		value := b.values[i]
		cleankey := b.cleankeys[i]

		if key == nil {
			if _, err := fmt.Fprint(b.w, string(value)); err != nil {
				return err
			}
			continue
		}
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

			padding := maxKeyLen - len(cleankey)
			line := []byte{}
			line = append(line, []byte(styleSeq.String())...)
			line = append(line, key...)
			line = append(line, []byte(resetSeq.String())...)
			line = append(line, ':')
			line = append(line, bytes.Repeat([]byte(" "), padding)...)
			line = append(line, ' ')
			line = append(line, value...)
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
