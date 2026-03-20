// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package uitui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

type keyBindings struct {
	Help key.Binding
	Back key.Binding
	Quit key.Binding
}

type helpKeyMap struct {
	short []key.Binding
	full  [][]key.Binding
}

func (k helpKeyMap) ShortHelp() []key.Binding {
	return k.short
}

func (k helpKeyMap) FullHelp() [][]key.Binding {
	return k.full
}

func defaultKeyBindings() keyBindings {
	return keyBindings{
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
	}
}

func newHelpModel() help.Model {
	m := help.New()
	m.Styles = helpStyles
	return m
}

func (m *Model) helpMap() helpKeyMap {
	panelActions := actionBindings(m.focusedActions())
	globalActions := actionBindings(m.globalActions())
	short := make([]key.Binding, 0, len(panelActions)+len(globalActions))
	short = append(short, panelActions...)
	short = append(short, globalActions...)

	full := make([][]key.Binding, 0, 2)
	if len(panelActions) > 0 {
		full = append(full, panelActions)
	}
	if len(globalActions) > 0 {
		full = append(full, globalActions)
	}

	return helpKeyMap{short: short, full: full}
}

func actionBindings(actions []Action) []key.Binding {
	bindings := make([]key.Binding, 0, len(actions))
	for _, action := range actions {
		if len(action.Keys) == 0 {
			continue
		}
		bindings = append(bindings, key.NewBinding(
			key.WithKeys(action.Keys...),
			key.WithHelp(action.Keys[0], action.Label),
		))
	}
	return bindings
}
