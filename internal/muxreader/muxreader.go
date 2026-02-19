// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package muxreader

import (
	"bufio"
	"fmt"
	"io"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"unikraft.com/x/colors"
)

type Mux struct {
	*io.PipeReader
	w      *io.PipeWriter
	mu     sync.Mutex
	width  int
	last   string
	sealed bool
	wg     sync.WaitGroup
	wOnce  sync.Once
}

var prefixStyle = lipgloss.NewStyle().Foreground(colors.Slate500).Faint(true)

func New() *Mux {
	r, w := io.Pipe()
	return &Mux{PipeReader: r, w: w}
}

func (m *Mux) With(name string, r io.Reader) {
	m.mu.Lock()
	if m.sealed {
		m.mu.Unlock()
		panic("called after Seal")
	}
	m.width = max(m.width, len(name))
	m.wg.Add(1)
	m.mu.Unlock()
	go m.run(name, r)
}

func (m *Mux) Seal() {
	m.mu.Lock()
	m.sealed = true
	m.mu.Unlock()
	go func() {
		m.wg.Wait()
		m.closeWriter(nil)
	}()
}

func (m *Mux) Close() {
	_ = m.PipeReader.Close()
	m.closeWriter(io.ErrClosedPipe)
}

func (m *Mux) closeWriter(err error) {
	m.wOnce.Do(func() {
		if err != nil {
			_ = m.w.CloseWithError(err)
		} else {
			_ = m.w.Close()
		}
	})
}

const (
	markerNew  = '┏'
	markerPipe = '│'
)

func (m *Mux) run(name string, r io.Reader) {
	defer m.wg.Done()
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) == 0 && err != nil {
			return
		}
		if err == io.EOF && len(line) > 0 && line[len(line)-1] != '\n' {
			line = append(line, '\n')
		}

		m.mu.Lock()
		switched := name != m.last
		m.last = name
		width := m.width
		m.mu.Unlock()

		marker := markerPipe
		if switched {
			marker = markerNew
		}
		prefix := prefixStyle.Render(fmt.Sprintf("%*s %c ", width, name, marker))

		_, werr := m.w.Write([]byte(prefix))
		if werr == nil {
			_, werr = m.w.Write(line)
		}
		if werr != nil {
			return
		}

		if err != nil {
			if err != io.EOF {
				m.closeWriter(err)
			}
			return
		}
	}
}
