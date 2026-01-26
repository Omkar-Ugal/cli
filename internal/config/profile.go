// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/MakeNowJust/heredoc"
	jujuerrors "github.com/juju/errors"
)

// DefaultProfileName is the name of the default profile used by the Unikraft
// CLI.  It is used when no specific profile is set or when the user has not
// created any profiles yet.
const DefaultProfileName = "default"

var (
	// ErrNoCurrentProfile is returned when there is no current profile set in the
	// configuration. This can happen if the user has not logged in or if the
	// current profile is not set in the configuration.
	ErrNoCurrentProfile = jujuerrors.New(heredoc.Docf(`
		profile not setup;

		use %[1]sunikraft login%[1]s to get started,

		or visit https://unikraft.com/docs/cli for more information`, "`"))
	// ErrProfileNotFound is returned when a profile with the specified name does
	// not exist in the configuration. This can happen if the user tries to access
	// a profile that has not been created or has been deleted.
	ErrProfileNotFound = jujuerrors.Errorf("profile not found")
)

// ProfileType represents the type of profile used in the Unikraft CLI.
// It can be either "local" for local profiles or "cloud" for cloud-based
// profiles.
type ProfileType string

const (
	// ProfileTypeLocal indicates a local profile, which is typically used for
	// development and testing purposes on the user's machine.
	ProfileTypeLocal ProfileType = "local"

	// ProfileTypeCloud indicates a cloud profile, which is used for accessing
	// cloud-based resources and services provided by Unikraft.
	ProfileTypeCloud ProfileType = "cloud"
)

// Profile represents a user profile configuration for the Unikraft CLI.
type Profile struct {
	Type         ProfileType `hidden:"" help:"Type of the profile." enum:"local,cloud" default:"cloud" json:"type" yaml:"type"`
	Name         string      `hidden:"" help:"Name of the profile." json:"name" yaml:"name"`
	Token        string      `hidden:"" help:"Authentication token for the profile." json:"token" yaml:"token"`
	Organization string      `hidden:"" help:"Organization associated with the profile." json:"organization,omitempty" yaml:"organization,omitempty"`
	ControlPlane string      `hidden:"" help:"Control plane endpoint for the profile." json:"controlplane,omitempty" yaml:"controlplane,omitempty"`
	Metros       []Metro     `hidden:"" embed:"" prefix:"metro." help:"Static list of metros." json:"metros,omitempty" yaml:"metros,omitempty"`
}

type Metro struct {
	Name     string `hidden:"" help:"Name of the metro." json:"name" yaml:"name"`
	Endpoint string `hidden:"" help:"Endpoint for the metro." json:"endpoint" yaml:"endpoint"`
	Country  string `hidden:"" help:"Country code for the metro." json:"country" yaml:"country"`
}

func (m Metro) Index() string {
	u, err := url.Parse(m.Endpoint)
	if err != nil {
		return m.Endpoint
	}
	hostname := u.Hostname()
	hostname, ok := strings.CutPrefix(hostname, "api.")
	if !ok {
		return m.Endpoint
	}
	return "index." + hostname
}

var _ fmt.Stringer = Profile{}

func (profile Profile) String() string {
	return profile.Name
}

// ProfileList returns a slice of profile names (aliases) from the
// configuration.
func (config *Config) ProfileList() []Profile {
	list := make([]Profile, 0, len(config.Profiles))
	for _, profile := range config.Profiles {
		list = append(list, profile)
	}
	return list
}

// CurrentProfile returns the currently selected profile from the configuration.
// If the profile does not exist, it returns an error.
func (config *Config) CurrentProfile() (*Profile, error) {
	profile, ok := config.Profiles[config.Profile]
	if !ok {
		if config.Profile == "" || config.Profile == DefaultProfileName {
			return nil, ErrNoCurrentProfile
		} else {
			return nil, ErrProfileNotFound
		}
	}

	return &profile, nil
}
