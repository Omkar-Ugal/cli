// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package types

import (
	"fmt"

	"unikraft.com/cloud/sdk/platform"
	"unikraft.com/x/colors"
)

// InstanceState is a wrapper around platform.InstanceState to automatically
// add pretty colors.
type InstanceState platform.InstanceState

func (state InstanceState) String() string {
	return state.color()(string(state))
}

func (state InstanceState) IsRunning() bool {
	switch platform.InstanceState(state) {
	case platform.InstanceStateRunning,
		platform.InstanceStateStarting,
		platform.InstanceStateStandby:
		return true
	default:
		return false
	}
}

func (state InstanceState) validate() error {
	switch platform.InstanceState(state) {
	case platform.InstanceStateStopped:
	case platform.InstanceStateStarting:
	case platform.InstanceStateRunning:
	case platform.InstanceStateDraining:
	case platform.InstanceStateStopping:
	case platform.InstanceStateStandby:
	default:
		return fmt.Errorf("unknown instance state: %q", string(state))
	}
	return nil
}

func (state *InstanceState) UnmarshalText(text []byte) error {
	s := InstanceState(text)
	if err := s.validate(); err != nil {
		return err
	}
	*state = s
	return nil
}

func (state InstanceState) MarshalText() ([]byte, error) {
	return []byte(state), nil
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

func (state VolumeState) validate() error {
	switch platform.VolumeState(state) {
	case platform.VolumeStateUninitialized:
	case platform.VolumeStateInitializing:
	case platform.VolumeStateAvailable:
	case platform.VolumeStateIdle:
	case platform.VolumeStateMounted:
	case platform.VolumeStateBusy:
	case platform.VolumeStateError:
	default:
		return fmt.Errorf("unknown volume state: %q", string(state))
	}
	return nil
}

func (state *VolumeState) UnmarshalText(text []byte) error {
	s := VolumeState(text)
	if err := s.validate(); err != nil {
		return err
	}
	*state = s
	return nil
}

func (state VolumeState) MarshalText() ([]byte, error) {
	return []byte(state), nil
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

func (state CertificateState) validate() error {
	switch platform.CertificateState(state) {
	case platform.CertificateStatePending:
	case platform.CertificateStateValid:
	case platform.CertificateStateError:
	default:
		return fmt.Errorf("unknown certificate state: %q", string(state))
	}
	return nil
}

func (state *CertificateState) UnmarshalText(text []byte) error {
	s := CertificateState(text)
	if err := s.validate(); err != nil {
		return err
	}
	*state = s
	return nil
}

func (state CertificateState) MarshalText() ([]byte, error) {
	return []byte(state), nil
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
