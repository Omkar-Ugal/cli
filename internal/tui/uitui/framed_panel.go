// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package uitui

import tea "charm.land/bubbletea/v2"

type TitleProvider interface {
	Title() string
}

// framedPanel renders a panel inside a standard frame and forwards content
// dimensions (excluding borders/padding/title) to the wrapped model.
type framedPanel struct {
	inner   tea.Model
	width   int
	height  int
	focused bool
}

func newFramedPanel(inner tea.Model) tea.Model {
	if inner == nil {
		return nil
	}
	return &framedPanel{inner: inner}
}

func (p *framedPanel) Init() tea.Cmd {
	return p.inner.Init()
}

func (p *framedPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		return p, p.forwardSize()
	case PanelFocusMsg:
		p.focused = msg.Focused
		updated, cmd := p.inner.Update(msg)
		p.inner = updated
		return p, cmd
	}

	updated, cmd := p.inner.Update(msg)
	p.inner = updated
	return p, cmd
}

func (p *framedPanel) View() tea.View {
	if p.inner == nil {
		return tea.NewView("")
	}

	body := p.inner.View().Content
	title := p.title()
	if title != "" {
		if body != "" {
			body = titleStyle.Render(title) + "\n" + body
		} else {
			body = titleStyle.Render(title)
		}
	}

	style := panelStyle
	if p.focused {
		style = panelFocusedStyle
	}
	if p.width > 0 {
		style = style.Width(p.width)
	}
	if p.height > 0 {
		style = style.Height(p.height)
	}

	return tea.NewView(style.Render(body))
}

func (p *framedPanel) Actions() []Action {
	if provider, ok := p.inner.(ActionProvider); ok {
		return provider.Actions()
	}
	return nil
}

func (p *framedPanel) Breadcrumb() string {
	if provider, ok := p.inner.(BreadcrumbProvider); ok {
		return provider.Breadcrumb()
	}
	return ""
}

func (p *framedPanel) Refresh() tea.Cmd {
	if provider, ok := p.inner.(RefreshProvider); ok {
		return provider.Refresh()
	}
	return nil
}

func (p *framedPanel) Close() error {
	if p.inner == nil {
		return nil
	}
	if c, ok := p.inner.(interface{ Close() error }); ok {
		return c.Close()
	}
	if c, ok := p.inner.(interface{ Close() }); ok {
		c.Close()
	}
	return nil
}

func (p *framedPanel) title() string {
	if provider, ok := p.inner.(TitleProvider); ok {
		return provider.Title()
	}
	return ""
}

func (p *framedPanel) forwardSize() tea.Cmd {
	if p.inner == nil {
		return nil
	}

	title := p.title()
	contentWidth := max(0, p.width-panelStyle.GetHorizontalFrameSize())
	contentHeight := max(0, p.height-panelStyle.GetVerticalFrameSize())
	if title != "" {
		contentHeight = max(0, contentHeight-1)
	}

	updated, cmd := p.inner.Update(tea.WindowSizeMsg{Width: contentWidth, Height: contentHeight})
	p.inner = updated
	return cmd
}

var (
	_ tea.Model          = (*framedPanel)(nil)
	_ ActionProvider     = (*framedPanel)(nil)
	_ BreadcrumbProvider = (*framedPanel)(nil)
	_ RefreshProvider    = (*framedPanel)(nil)
)
