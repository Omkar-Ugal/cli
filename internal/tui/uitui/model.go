// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package uitui

import (
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	refreshInterval = time.Second
	minPanelWidth   = 50
	panelGapWidth   = 1
	panelWidthRatio = 1.5 // each newer panel is this much wider than the previous
)

// Action types for global actions
type (
	helpAction struct{}
	backAction struct{}
	quitAction struct{}
)

// panelMsg is a wrapper for messages coming from / directed to a specific
// panel, allowing us to route them correctly in the main update loop.
type panelMsg struct {
	tea.Msg
	idx int
}

type refreshTickMsg struct{}

type panelLayout struct {
	index int
	width int
}

type Model struct {
	panels    []tea.Model
	collapsed []bool
	width     int
	height    int
	help      help.Model
	keys      keyBindings
}

func NewModel(panels ...tea.Model) *Model {
	// Wrap each panel with the standard UI wrappers.
	wrapped := make([]tea.Model, len(panels))
	for i, p := range panels {
		wrapped[i] = wrapPanel(p)
	}
	return &Model{
		panels:    wrapped,
		collapsed: make([]bool, len(wrapped)),
		help:      newHelpModel(),
		keys:      defaultKeyBindings(),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.initPanelsCmd(),
		m.refreshVisibleCmd(),
		refreshTickCmd(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)
		return m, m.syncLayout()
	case refreshTickMsg:
		return m, tea.Batch(
			m.refreshVisibleCmd(),
			refreshTickCmd(),
		)
	case OpenPanelMsg:
		return m, m.openPanel(msg.Panel, msg.Collapse)
	case panelMsg:
		return m, m.handlePanelMsg(msg)
	case tea.KeyMsg:
		return m, m.handleKeyMsg(msg)
	}

	return m, nil
}

func (m *Model) View() tea.View {
	if len(m.panels) == 0 {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}

	helpView := m.help.View(m.helpMap())
	helpView = strings.TrimRight(helpView, "\n")
	breadcrumbView := m.breadcrumbView()
	breadcrumbView = strings.TrimRight(breadcrumbView, "\n")
	content := m.renderPanels()

	parts := make([]string, 0, 3)
	if breadcrumbView != "" {
		parts = append(parts, breadcrumbView)
	}
	if content != "" {
		parts = append(parts, content)
	}
	if helpView != "" {
		parts = append(parts, helpContainerStyle.Render(helpView))
	}
	if len(parts) == 0 {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	var v tea.View
	if len(parts) == 1 {
		v = tea.NewView(parts[0])
	} else {
		v = tea.NewView(lipgloss.JoinVertical(lipgloss.Left, parts...))
	}
	v.AltScreen = true
	return v
}

func (m *Model) handlePanelMsg(msg panelMsg) tea.Cmd {
	if msg.idx < 0 || msg.idx >= len(m.panels) {
		return nil
	}

	switch inner := msg.Msg.(type) {
	case OpenPanelMsg:
		return m.openPanel(inner.Panel, inner.Collapse)
	default:
		cmd := m.applyPanelMsg(msg.idx, inner)
		return tea.Batch(cmd, m.syncLayout())
	}
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	if len(m.panels) == 0 {
		return nil
	}

	if action := actionForKey(msg.String(), m.keyActions()); action != nil {
		return m.applyAction(*action)
	}

	return m.applyPanelMsg(m.focusedIndex(), msg)
}

func (m *Model) focusedIndex() int {
	if len(m.panels) == 0 {
		return -1
	}
	return len(m.panels) - 1
}

func (m *Model) visibleCountFor(renderableCount int) int {
	if renderableCount == 0 {
		return 0
	}
	if m.width <= 0 {
		return 1
	}
	count := max(1, m.width/minPanelWidth)
	return min(count, renderableCount)
}

func (m *Model) renderableIndices() []int {
	if len(m.panels) == 0 {
		return nil
	}
	indices := make([]int, 0, len(m.panels))
	for i := range m.panels {
		if i < len(m.collapsed) && m.collapsed[i] {
			continue
		}
		indices = append(indices, i)
	}
	return indices
}

func (m *Model) visibleIndices() []int {
	renderable := m.renderableIndices()
	count := m.visibleCountFor(len(renderable))
	if count == 0 {
		return nil
	}
	start := len(renderable) - count
	return renderable[start:]
}

func (m *Model) renderPanels() string {
	if len(m.panels) == 0 {
		return ""
	}

	visible := m.visibleIndices()
	if len(visible) == 0 {
		return ""
	}
	if len(visible) == 1 {
		idx := visible[0]
		return m.panels[idx].View().Content
	}

	gap := strings.Repeat(" ", panelGapWidth)
	parts := make([]string, 0, len(visible)*2-1)
	for i, idx := range visible {
		if i > 0 {
			parts = append(parts, gap)
		}
		parts = append(parts, m.panels[idx].View().Content)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m *Model) refreshVisibleCmd() tea.Cmd {
	visible := m.visibleIndices()
	if len(visible) == 0 {
		return nil
	}

	cmds := make([]tea.Cmd, 0, len(visible))
	for _, idx := range visible {
		cmds = append(cmds, m.refreshPanelCmd(idx))
	}
	return tea.Batch(cmds...)
}

func (m *Model) refreshPanelCmd(index int) tea.Cmd {
	if index < 0 || index >= len(m.panels) {
		return nil
	}
	refreshable, ok := m.panels[index].(RefreshProvider)
	if !ok {
		return nil
	}
	return wrapPanelCmd(index, refreshable.Refresh())
}

func (m *Model) applyPanelMsg(index int, msg tea.Msg) tea.Cmd {
	if index < 0 || index >= len(m.panels) {
		return nil
	}
	updated, cmd := m.panels[index].Update(msg)
	if updated != nil {
		m.panels[index] = updated
	}
	return wrapPanelCmd(index, cmd)
}

func (m *Model) applyAction(action Action) tea.Cmd {
	switch action.Value.(type) {
	case helpAction:
		m.help.ShowAll = !m.help.ShowAll
		return m.syncLayout()
	case backAction:
		if len(m.panels) > 1 {
			closeTeaModel(m.panels[len(m.panels)-1])
			m.panels = m.panels[:len(m.panels)-1]
			if len(m.collapsed) > len(m.panels) {
				m.collapsed = m.collapsed[:len(m.panels)]
			}
			if len(m.collapsed) > 0 {
				m.collapsed[len(m.collapsed)-1] = false
			}
			return m.syncLayout()
		}
		if len(m.panels) == 1 {
			closeTeaModel(m.panels[0])
		}
		return tea.Quit
	case quitAction:
		for _, p := range m.panels {
			closeTeaModel(p)
		}
		return tea.Quit
	}
	if action.Value == nil {
		return nil
	}
	return m.applyPanelMsg(m.focusedIndex(), action.Value)
}

func (m *Model) focusedActions() []Action {
	return m.panelActions(m.focusedIndex())
}

func (m *Model) panelActions(index int) []Action {
	if index < 0 || index >= len(m.panels) {
		return nil
	}
	provider, ok := m.panels[index].(ActionProvider)
	if !ok {
		return nil
	}
	return provider.Actions()
}

func (m *Model) globalActions() []Action {
	return []Action{
		actionFromBinding(backAction{}, m.keys.Back),
		actionFromBinding(helpAction{}, m.keys.Help),
		actionFromBinding(quitAction{}, m.keys.Quit),
	}
}

func (m *Model) keyActions() []Action {
	panelActions := m.focusedActions()
	globalActions := m.globalActions()
	actions := make([]Action, 0, len(panelActions)+len(globalActions))
	actions = append(actions, globalActions...)
	actions = append(actions, panelActions...)
	return actions
}

func (m *Model) openPanel(panel tea.Model, collapse bool) tea.Cmd {
	if panel == nil {
		return nil
	}
	panel = wrapPanel(panel)

	if len(m.collapsed) < len(m.panels) {
		m.collapsed = append(m.collapsed, make([]bool, len(m.panels)-len(m.collapsed))...)
	}
	if collapse && len(m.panels) > 0 {
		m.collapsed[len(m.panels)-1] = true
	}
	m.panels = append(m.panels, panel)
	m.collapsed = append(m.collapsed, false)
	index := len(m.panels) - 1
	return tea.Batch(
		m.initPanelCmd(index),
		m.refreshPanelCmd(index),
		m.syncLayout(),
	)
}

func (m *Model) initPanelsCmd() tea.Cmd {
	if len(m.panels) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(m.panels))
	for i := range m.panels {
		cmds = append(cmds, m.initPanelCmd(i))
	}
	return tea.Batch(cmds...)
}

func (m *Model) initPanelCmd(index int) tea.Cmd {
	if index < 0 || index >= len(m.panels) {
		return nil
	}
	return wrapPanelCmd(index, m.panels[index].Init())
}

func refreshTickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func (m *Model) mainHeight() int {
	helpView := m.help.View(m.helpMap())
	helpView = strings.TrimRight(helpView, "\n")
	breadcrumbView := m.breadcrumbView()
	breadcrumbView = strings.TrimRight(breadcrumbView, "\n")
	helpLines := 0
	if helpView != "" {
		helpLines = 1 + strings.Count(helpView, "\n")
	}
	breadcrumbLines := 0
	if breadcrumbView != "" {
		breadcrumbLines = 1 + strings.Count(breadcrumbView, "\n")
	}
	return max(0, m.height-helpLines-breadcrumbLines)
}

func (m *Model) breadcrumbTrail() []string {
	if len(m.panels) == 0 {
		return nil
	}
	trail := make([]string, 0, len(m.panels))
	for _, panel := range m.panels {
		provider, ok := panel.(BreadcrumbProvider)
		if !ok {
			continue
		}
		crumb := strings.TrimSpace(provider.Breadcrumb())
		if crumb == "" {
			continue
		}
		trail = append(trail, crumb)
	}
	return trail
}

func (m *Model) breadcrumbView() string {
	trail := m.breadcrumbTrail()
	if len(trail) == 0 || m.width <= 0 {
		return ""
	}

	separator := breadcrumbSeparatorStyle.Render(" > ")
	parts := make([]string, 0, len(trail)*2-1)
	last := len(trail) - 1
	for i, crumb := range trail {
		if i > 0 {
			parts = append(parts, separator)
		}
		style := breadcrumbStyle
		if i == last {
			style = breadcrumbCurrentStyle
		}
		parts = append(parts, style.Render(crumb))
	}
	line := strings.Join(parts, "")
	available := m.width - breadcrumbContainerStyle.GetHorizontalFrameSize()
	if available <= 0 {
		return ""
	}
	line = ansi.Truncate(line, available, "...")
	if line == "" {
		return ""
	}
	return breadcrumbContainerStyle.Render(line)
}

func (m *Model) visibleLayouts() []panelLayout {
	visible := m.visibleIndices()
	if len(visible) == 0 {
		return nil
	}

	contentWidth := max(0, m.width-panelGapWidth*(len(visible)-1))

	if len(visible) == 1 {
		return []panelLayout{{index: visible[0], width: contentWidth}}
	}

	widths := geometricWidths(contentWidth, len(visible), panelWidthRatio)
	layouts := make([]panelLayout, 0, len(visible))
	for i, idx := range visible {
		width := 0
		if i < len(widths) {
			width = widths[i]
		}
		layouts = append(layouts, panelLayout{index: idx, width: width})
	}
	return layouts
}

func geometricWidths(total, n int, ratio float64) []int {
	if n <= 0 {
		return nil
	}
	widths := make([]int, n)
	if n == 1 {
		widths[0] = total
		return widths
	}
	if total <= 0 {
		return widths
	}
	if ratio <= 1 {
		each := total / n
		assigned := 0
		for i := range n {
			widths[i] = each
			assigned += each
		}
		widths[n-1] += total - assigned
		return widths
	}

	// For n panels with ratio r, if the smallest has width w:
	//   total = w + w*r + w*r^2 + ... + w*r^(n-1) = w * (r^n - 1) / (r - 1)
	// So: w = total * (r - 1) / (r^n - 1)
	rn := 1.0
	for range n {
		rn *= ratio
	}
	smallestWidth := float64(total) * (ratio - 1) / (rn - 1)

	assigned := 0
	multiplier := 1.0
	for i := range n {
		w := int(smallestWidth * multiplier)
		if i == n-1 {
			w = total - assigned
		}
		widths[i] = w
		assigned += w
		multiplier *= ratio
	}
	return widths
}

func (m *Model) syncLayout() tea.Cmd {
	if len(m.panels) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(m.panels))
	focused := m.focusedIndex()
	for i := range m.panels {
		cmds = append(cmds, m.applyPanelMsg(i, PanelFocusMsg{Focused: i == focused}))
	}

	height := m.mainHeight()
	for _, layout := range m.visibleLayouts() {
		cmds = append(cmds, m.applyPanelMsg(layout.index, tea.WindowSizeMsg{Width: layout.width, Height: height}))
	}
	return tea.Batch(cmds...)
}

func wrapPanelCmd(index int, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if msg == nil {
			return nil
		}
		// Don't wrap BatchMsg - wrap each command in the batch instead
		if batch, ok := msg.(tea.BatchMsg); ok {
			wrapped := make([]tea.Cmd, len(batch))
			for i, c := range batch {
				wrapped[i] = wrapPanelCmd(index, c)
			}
			return tea.BatchMsg(wrapped)
		}
		return panelMsg{idx: index, Msg: msg}
	}
}

func actionForKey(key string, actions []Action) *Action {
	for _, action := range actions {
		if slices.Contains(action.Keys, key) {
			copy := action
			return &copy
		}
	}
	return nil
}

func actionFromBinding(value any, binding key.Binding) Action {
	return Action{
		Label: binding.Help().Desc,
		Keys:  binding.Keys(),
		Value: value,
	}
}

func closeTeaModel(m tea.Model) {
	if m == nil {
		return
	}
	if c, ok := m.(interface{ Close() error }); ok {
		_ = c.Close()
		return
	}
	if c, ok := m.(interface{ Close() }); ok {
		c.Close()
	}
}
