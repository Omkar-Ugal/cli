// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package uitui

import tea "charm.land/bubbletea/v2"

type Action struct {
	Label string
	Keys  []string
	Value any
}

type ActionMsg = any

type OpenPanelMsg struct {
	Panel    tea.Model
	Collapse bool
}

type ActionProvider interface {
	Actions() []Action
}

type BreadcrumbProvider interface {
	Breadcrumb() string
}

type RefreshProvider interface {
	Refresh() tea.Cmd
}

// SubpanelsProvider is implemented by panels that have subpanels displayed below them.
// Subpanels divide the available height with the main panel and can be focused via tab/shift-tab.
type SubpanelsProvider interface {
	// Subpanels returns the list of subpanels to display below this panel.
	Subpanels() []tea.Model
}

type PanelFocusMsg struct {
	Focused bool
}
