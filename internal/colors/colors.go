// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package colors

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	Blue500    = lipgloss.Color("#2b7fff")
	Emerald500 = lipgloss.Color("#00bc7d")
	Orange500  = lipgloss.Color("#ff6900")
	Rose600    = lipgloss.Color("#ec003f")
	Slate400   = lipgloss.Color("#90a1b9")

	Primary     = lipgloss.AdaptiveColor{Light: string(Blue500), Dark: string(Blue500)}
	PrimaryFg   = lipgloss.NewStyle().Foreground(Primary).Render
	PrimaryFgBg = lipgloss.NewStyle().Background(Primary).Foreground(Primary).Render
	Success     = lipgloss.AdaptiveColor{Light: string(Emerald500), Dark: string(Emerald500)}
	SuccessFg   = lipgloss.NewStyle().Foreground(Success).Render
	SuccessFgBg = lipgloss.NewStyle().Background(Success).Foreground(Success).Render
	Warning     = lipgloss.AdaptiveColor{Light: string(Orange500), Dark: string(Orange500)}
	WarningFg   = lipgloss.NewStyle().Foreground(Warning).Render
	WarningFgBg = lipgloss.NewStyle().Background(Warning).Foreground(Warning).Render
	Error       = lipgloss.AdaptiveColor{Light: string(Rose600), Dark: string(Rose600)}
	ErrorFg     = lipgloss.NewStyle().Foreground(Error).Render
	ErrorFgBg   = lipgloss.NewStyle().Background(Error).Foreground(Error).Render
	Info        = lipgloss.AdaptiveColor{Light: string(Slate400), Dark: string(Slate400)}
	InfoFg      = lipgloss.NewStyle().Foreground(Info).Render
	InfoFgBg    = lipgloss.NewStyle().Background(Info).Foreground(Info).Render
)
