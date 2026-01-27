// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package types

import (
	"unikraft.com/cloud/sdk/platform"
	"unikraft.com/x/colors"
)

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
