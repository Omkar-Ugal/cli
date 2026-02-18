// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package types

import (
	"encoding/json"
	"fmt"
	"time"

	"unikraft.com/x/colors"
)

// MetroStatus represents the connection status of a metro endpoint.
type MetroStatus string

const (
	MetroStatusOnline  MetroStatus = "online"
	MetroStatusOffline MetroStatus = "offline"
)

func (s MetroStatus) String() string {
	return s.color()(string(s))
}

func (s MetroStatus) color() func(...string) string {
	switch s {
	case MetroStatusOnline:
		return colors.SuccessFg
	case MetroStatusOffline:
		return colors.ErrorFg
	}
	return colors.InfoFg
}

// PingLatency represents network latency to a metro endpoint.
type PingLatency time.Duration

func (p PingLatency) String() string {
	dur := time.Duration(p)
	return p.color()(formatDuration(dur))
}

func (p PingLatency) MarshalText() ([]byte, error) {
	return []byte(formatDuration(time.Duration(p))), nil
}

func (p PingLatency) MarshalJSON() ([]byte, error) {
	text, err := p.MarshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(text))
}

func (p PingLatency) color() func(...string) string {
	dur := time.Duration(p)
	switch {
	case dur < 100*time.Millisecond:
		return colors.SuccessFg
	case dur < 250*time.Millisecond:
		return colors.WarningFg
	default:
		return colors.ErrorFg
	}
}

// formatDuration formats the duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return d.Round(time.Millisecond).String()
}
