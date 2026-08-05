// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package types

import (
	"cmp"
	"fmt"

	"unikraft.com/cli/internal/resource/value"
	"unikraft.com/cli/internal/tui/meter"
)

// Usage represents a used/limit pair for quota tracking.
// It marshals to text as "used/limit".
type Usage[T any] struct {
	Used  T `field:",long"`
	Limit T `field:",long"`
}

func (u Usage[T]) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%v/%v", u.Used, u.Limit), nil
}

func (u Usage[T]) String() string {
	return fmt.Sprintf("%v/%v", u.Used, u.Limit)
}

func (u Usage[T]) Value() any {
	return u
}

// Range represents a min/max range for quota limits.
// It marshals to text as "min...max".
type Range[T any] struct {
	Min T `field:",long"`
	Max T `field:",long"`
}

func (r Range[T]) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%v...%v", r.Min, r.Max), nil
}

func (r Range[T]) String() string {
	return fmt.Sprintf("%v...%v", r.Min, r.Max)
}

func (r Range[T]) Value() any {
	return r
}

type numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// MeterUsage represents a used/total pair for quotas, rendered as a colored
// meter bar.
type MeterUsage[T numeric] struct {
	Used  T `field:",long"`
	Total T `field:",long"`
}

// Compare compares two MeterUsage values by their used/total ratio,
// returning -1, 0, or 1 per the same contract as cmp.Compare.
func (u MeterUsage[T]) Compare(other MeterUsage[T]) int {
	return cmp.Compare(u.ratio(), other.ratio())
}

// ratio returns the used/total ratio, or 0 if Total is 0.
func (u MeterUsage[T]) ratio() float64 {
	if u.Total == 0 {
		return 0
	}
	return float64(u.Used) / float64(u.Total)
}

// clampedRatio returns the used/total ratio clamped to [0, 1], or 0 if Total
// is 0.
func (u MeterUsage[T]) clampedRatio() float64 {
	ratio := u.ratio()
	if ratio > 1 {
		return 1
	}
	if ratio < 0 {
		return 0
	}
	return ratio
}

// String returns the used/total ratio as a raw percentage, e.g. "80%", or
// "" if Total is 0.
func (u MeterUsage[T]) String() string {
	if u.Total == 0 {
		return ""
	}
	return fmt.Sprintf("%.0f%%", u.clampedRatio()*100)
}

// Render implements value.Renderer, rendering the usage as a percentage
// followed by a colored meter bar, e.g. "80% ⣿⣿⣿⣿⣿⣿⣿⣿⣀⣀". In quiet mode
// it renders just the raw percentage, matching String.
func (u MeterUsage[T]) Render(opts value.RenderOpts) (string, error) {
	if opts.Quiet {
		return u.String(), nil
	}
	if u.Total == 0 {
		return "N/A", nil
	}

	bar := meter.Render(u.clampedRatio(), meter.Width())

	// Right-pad the percentage to the width of "100%" so the bar stays on
	// the same column regardless of how many digits the percentage has.
	pct := fmt.Sprintf("%-4s", u.String())

	return fmt.Sprintf("%s %s", pct, bar), nil
}

func (u MeterUsage[T]) MarshalText() ([]byte, error) {
	if u.Total == 0 {
		return []byte("N/A"), nil
	}
	return fmt.Appendf(nil, "%v/%v", u.Used, u.Total), nil
}
