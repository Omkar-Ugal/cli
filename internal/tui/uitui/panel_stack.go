// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package uitui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	subpanelGapHeight = 0
	minSubpanelHeight = 8
)

// wrapPanel applies the standard UI wrappers:
// - frame handling (borders/padding/title)
// - subpanel stacking (each subpanel is framed too)
func wrapPanel(panel tea.Model) tea.Model {
	if panel == nil {
		return nil
	}
	provider, ok := panel.(SubpanelsProvider)
	if !ok {
		return newFramedPanel(panel)
	}
	subpanels := provider.Subpanels()
	if len(subpanels) == 0 {
		return newFramedPanel(panel)
	}
	framedSubpanels := make([]tea.Model, 0, len(subpanels))
	for _, sp := range subpanels {
		framedSubpanels = append(framedSubpanels, newFramedPanel(sp))
	}
	return &panelStack{
		main:       newFramedPanel(panel),
		subpanels:  framedSubpanels,
		focusIndex: 0, // main panel is focused by default
	}
}

// panelStack wraps a main panel and its subpanels, managing focus between them.
// It handles tab/shift-tab for focus cycling and renders panels vertically.
type panelStack struct {
	main       tea.Model
	subpanels  []tea.Model
	focusIndex int // -1 means not focused, 0 = main panel, 1+ = subpanel index + 1

	width   int
	height  int
	focused bool
}

func (ps *panelStack) Init() tea.Cmd {
	cmds := []tea.Cmd{ps.main.Init()}
	for _, sp := range ps.subpanels {
		cmds = append(cmds, sp.Init())
	}
	return tea.Batch(cmds...)
}

func (ps *panelStack) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ps.width = msg.Width
		ps.height = msg.Height
		return ps, ps.syncLayout()

	case PanelFocusMsg:
		ps.focused = msg.Focused
		if !ps.focused {
			ps.focusIndex = 0
		}
		return ps, ps.syncLayout()

	case tea.KeyMsg:
		if ps.focused {
			switch msg.String() {
			case "tab":
				return ps, ps.cycleFocusForward()
			case "shift+tab":
				return ps, ps.cycleFocusBackward()
			}
		}
		return ps, ps.forwardToFocused(msg)
	}

	// Forward other messages to all panels (each panel only handles its own message types).
	return ps, ps.forwardToAll(msg)
}

func (ps *panelStack) View() tea.View {
	if ps.height <= 0 || ps.width <= 0 {
		return tea.NewView("")
	}

	heights := ps.calculateHeights()
	parts := make([]string, 0, len(ps.subpanels)+1)

	parts = append(parts, ps.main.View().Content)
	for i, sp := range ps.subpanels {
		if i < len(heights)-1 && heights[i+1] > 0 {
			parts = append(parts, sp.View().Content)
		}
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (ps *panelStack) Breadcrumb() string {
	if provider, ok := ps.main.(BreadcrumbProvider); ok {
		return provider.Breadcrumb()
	}
	return ""
}

func (ps *panelStack) Actions() []Action {
	var focused tea.Model
	if ps.focusIndex == 0 {
		focused = ps.main
	} else if ps.focusIndex > 0 && ps.focusIndex <= len(ps.subpanels) {
		focused = ps.subpanels[ps.focusIndex-1]
	}

	if focused == nil {
		return nil
	}
	if provider, ok := focused.(ActionProvider); ok {
		return provider.Actions()
	}
	return nil
}

func (ps *panelStack) Refresh() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(ps.subpanels)+1)
	if provider, ok := ps.main.(RefreshProvider); ok {
		cmds = append(cmds, provider.Refresh())
	}
	for _, sp := range ps.subpanels {
		if provider, ok := sp.(RefreshProvider); ok {
			cmds = append(cmds, provider.Refresh())
		}
	}
	return tea.Batch(cmds...)
}

func (ps *panelStack) Close() error {
	closeTeaModel(ps.main)
	for _, sp := range ps.subpanels {
		closeTeaModel(sp)
	}
	return nil
}

func (ps *panelStack) calculateHeights() []int {
	if ps.height <= 0 {
		return nil
	}

	numPanels := 1 + len(ps.subpanels)
	heights := make([]int, numPanels)

	totalGaps := subpanelGapHeight * len(ps.subpanels)
	availableHeight := max(0, ps.height-totalGaps)

	if len(ps.subpanels) == 0 {
		heights[0] = availableHeight
		return heights
	}

	mainHeight := availableHeight / 2
	subHeight := (availableHeight - mainHeight) / len(ps.subpanels)

	if subHeight < minSubpanelHeight && availableHeight > minSubpanelHeight*numPanels {
		subHeight = minSubpanelHeight
		mainHeight = availableHeight - subHeight*len(ps.subpanels)
	}

	heights[0] = mainHeight
	for i := 1; i < numPanels; i++ {
		heights[i] = subHeight
	}

	remainder := availableHeight - mainHeight - subHeight*len(ps.subpanels)
	for i := 0; i < remainder && i < numPanels; i++ {
		heights[i]++
	}

	return heights
}

func (ps *panelStack) syncLayout() tea.Cmd {
	heights := ps.calculateHeights()
	if len(heights) == 0 {
		return nil
	}

	cmds := make([]tea.Cmd, 0, len(ps.subpanels)+1)

	mainFocused := ps.focused && ps.focusIndex == 0
	updated, cmd := ps.main.Update(PanelFocusMsg{Focused: mainFocused})
	ps.main = updated
	cmds = append(cmds, cmd)

	updated, cmd = ps.main.Update(tea.WindowSizeMsg{Width: ps.width, Height: heights[0]})
	ps.main = updated
	cmds = append(cmds, cmd)

	for i, sp := range ps.subpanels {
		subFocused := ps.focused && ps.focusIndex == i+1
		updated, cmd := sp.Update(PanelFocusMsg{Focused: subFocused})
		ps.subpanels[i] = updated
		cmds = append(cmds, cmd)

		if i+1 < len(heights) {
			updated, cmd = ps.subpanels[i].Update(tea.WindowSizeMsg{Width: ps.width, Height: heights[i+1]})
			ps.subpanels[i] = updated
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

func (ps *panelStack) cycleFocusForward() tea.Cmd {
	totalPanels := 1 + len(ps.subpanels)
	ps.focusIndex = (ps.focusIndex + 1) % totalPanels
	return ps.syncLayout()
}

func (ps *panelStack) cycleFocusBackward() tea.Cmd {
	totalPanels := 1 + len(ps.subpanels)
	ps.focusIndex = (ps.focusIndex - 1 + totalPanels) % totalPanels
	return ps.syncLayout()
}

func (ps *panelStack) forwardToFocused(msg tea.Msg) tea.Cmd {
	var focused tea.Model
	if ps.focusIndex == 0 {
		focused = ps.main
	} else if ps.focusIndex > 0 && ps.focusIndex <= len(ps.subpanels) {
		focused = ps.subpanels[ps.focusIndex-1]
	}

	if focused == nil {
		return nil
	}

	updated, cmd := focused.Update(msg)
	if ps.focusIndex == 0 {
		ps.main = updated
	} else {
		ps.subpanels[ps.focusIndex-1] = updated
	}
	return cmd
}

func (ps *panelStack) forwardToAll(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(ps.subpanels)+1)

	updated, cmd := ps.main.Update(msg)
	ps.main = updated
	cmds = append(cmds, cmd)

	for i, sp := range ps.subpanels {
		updated, cmd := sp.Update(msg)
		ps.subpanels[i] = updated
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}
