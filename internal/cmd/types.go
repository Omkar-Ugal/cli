// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"time"

	"github.com/distribution/reference"
	"github.com/docker/go-units"
	"unikraft.com/cloud/sdk/platform"
	"unikraft.com/x/colors"
)

// DurationMS is a time wrapper that represents a duration in milliseconds.
type DurationMS int64

func (d DurationMS) Unwrap() any {
	return time.Duration(d) * time.Millisecond
}

// DurationUS is a time wrapper that represents a duration in microseconds.
type DurationUS int64

func (d DurationUS) Unwrap() any {
	return time.Duration(d) * time.Microsecond
}

// SizeMB is a size wrapper that represents a size in megabytes.
type SizeMB int64

func (s SizeMB) String() string {
	return units.HumanSize(units.MB * float64(s))
}

type ImageRef[T interface {
	reference.Reference
	comparable
}] struct {
	Reference T
}

func (ir ImageRef[T]) String() string {
	var zero T
	if ir.Reference == zero {
		return ""
	}
	return reference.FamiliarString(ir.Reference)
}

// InstanceState is a wrapper around platform.InstanceState to automatically
// add pretty colors.
type InstanceState platform.InstanceState

func (state InstanceState) String() string {
	return state.color()(string(state))
}

func (state InstanceState) color() func(...string) string {
	switch platform.InstanceState(state) {
	case platform.InstanceStateStopped:
		return colors.ErrorFg
	case platform.InstanceStateStarting:
		return colors.InfoFg
	case platform.InstanceStateRunning:
		return colors.SuccessFg
	case platform.InstanceStateDraining:
		return colors.WarningFg
	case platform.InstanceStateStopping:
		return colors.WarningFg
	case platform.InstanceStateStandby:
		return colors.PrimaryFg
	}
	return colors.InfoFg
}

type VolumeState platform.VolumeState

func (state VolumeState) String() string {
	return state.color()(string(state))
}

func (state VolumeState) color() func(...string) string {
	// FIXME: these colors probably aren't right
	switch platform.VolumeState(state) {
	case platform.VolumeStateUninitialized:
		return colors.InfoFg
	case platform.VolumeStateInitializing:
		return colors.WarningFg
	case platform.VolumeStateAvailable:
		return colors.SuccessFg
	case platform.VolumeStateIdle:
		return colors.PrimaryFg
	case platform.VolumeStateMounted:
		return colors.SuccessFg
	case platform.VolumeStateBusy:
		return colors.WarningFg
	case platform.VolumeStateError:
		return colors.ErrorFg
	}
	return colors.InfoFg
}

type CertificateState platform.CertificateState

func (state CertificateState) String() string {
	return state.color()(string(state))
}

func (state CertificateState) color() func(...string) string {
	switch platform.CertificateState(state) {
	case platform.CertificateStatePending:
		return colors.WarningFg
	case platform.CertificateStateValid:
		return colors.SuccessFg
	case platform.CertificateStateError:
		return colors.ErrorFg
	}
	return colors.InfoFg
}
