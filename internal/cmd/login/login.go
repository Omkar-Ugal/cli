// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package login

import (
	"context"
	"time"

	jujuerrors "github.com/juju/errors"
	"github.com/pkg/browser"
	"unikraft.com/cloud/sdk/controlplane"
	"unikraft.com/x/log"
	"unikraft.com/x/ptr"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/httpclient"
)

type LoginCmd struct {
	Check   bool          `name:"check" help:"Check if the user is already logged in."`
	Timeout time.Duration `short:"t" name:"timeout" default:"5m" help:"Timeout for the login request."`

	ControlPlane  string `name:"controlplane" default:"https://controlplane.unikraft.cloud" help:"Control plane URL to use for login."`
	AllowInsecure bool   `name:"allow-insecure" short:"k" help:"Allow insecure server connections when using SSL."`

	NoBrowser bool `name:"no-browser" help:"Do not open the browser automatically for login."`
}

func (cmd *LoginCmd) Run(cfg *config.Config) error {
	ctx := cfg.Context

	profile, err := config.G(ctx).CurrentProfile()
	if err != nil && jujuerrors.Is(err, config.ErrNoCurrentProfile) {
		// Set up a new profile if no current profile exists.
		profile = &config.Profile{
			Type:         config.ProfileTypeCloud,
			Name:         config.DefaultProfileName,
			ControlPlane: cmd.ControlPlane,
		}
	} else if err != nil && jujuerrors.Is(err, config.ErrProfileNotFound) {
		// Set up a new profile for the new profile.
		profile = &config.Profile{
			Type:         config.ProfileTypeCloud,
			Name:         cfg.Profile,
			ControlPlane: cmd.ControlPlane,
		}
	} else if err != nil {
		return jujuerrors.Annotate(err, "getting current profile")
	}

	if cmd.Check {
		if profile.Token != "" {
			log.G(ctx).Info().
				Msg("existing authentication token found")
			return nil
		}
		return jujuerrors.Errorf("no existing authentication token found")
	}

	if profile.Token != "" {
		log.G(ctx).Info().
			Msg("existing authentication token found, re-authenticating")
	}

	resp, err := cmd.getAuth(ctx, profile)
	if err != nil {
		return jujuerrors.Annotate(err, "getting authentication token")
	}
	if resp.Status == string(controlplane.ResponseStatusERROR) {
		return jujuerrors.Annotate(jujuerrors.New(resp.Message), "authentication failed")
	}
	if resp.Data == nil {
		return jujuerrors.New("no data received from control plane, please try again")
	}
	if resp.Data.Token == nil {
		return jujuerrors.New("no authentication token received from control plane, please try again")
	}

	profile.Token = *resp.Data.Token
	profile.Organization = ptr.ZeroIfNil(resp.Data.OrganizationName)
	profile.Metros = nil

	// Instantiate a new client to fetch the list of metros.
	client := controlplane.NewClient(
		controlplane.WithDefaultEndpoint(profile.ControlPlane),
		controlplane.WithToken(profile.Token),
	)

	metroResp, err := client.ListMetros(ctx)
	if err != nil {
		log.G(ctx).
			Warn().
			Err(err).
			Msg("could not list metros for profile: please add metros manually")
	} else if metroResp.Data == nil {
		log.G(ctx).
			Warn().
			Msg("could not list metros for profile: please add metros manually")
	}

	metros := ptr.ZeroIfNil(metroResp.Data)
	for _, metro := range metros.Metros {
		profile.Metros = append(profile.Metros, config.Metro{
			Name:     ptr.ZeroIfNil(metro.Name),
			Endpoint: ptr.ZeroIfNil(metro.Endpoint),
			Country:  ptr.ZeroIfNil(metro.Country),
		})
	}

	cfg.Profile = profile.Name
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.Profile)
	}
	cfg.Profiles[profile.Name] = *profile

	if err := cfg.Save(); err != nil {
		return jujuerrors.Annotate(err, "saving profile")
	}

	log.G(ctx).Info().
		Msg("login successful")
	return nil
}

func (cmd *LoginCmd) getAuth(ctx context.Context, profile *config.Profile) (*controlplane.Response[controlplane.CheckAuthorizationResponseData], error) {
	server := profile.ControlPlane
	if len(cmd.ControlPlane) > 0 {
		// Override the control plane if one is provided via the command line.
		server = cmd.ControlPlane
	} else if len(server) == 0 {
		// If no control plane is set, use the default control plane.
		server = controlplane.DefaultEndpoint
	}

	copts := []controlplane.ClientOption{
		controlplane.WithDefaultEndpoint(server),
	}

	if cmd.AllowInsecure {
		copts = append(copts, controlplane.WithHTTPClient(httpclient.InsecureHTTPClient))
	} else {
		copts = append(copts, controlplane.WithHTTPClient(httpclient.DefaultHTTPClient))
	}

	client := controlplane.NewClient(copts...)

	signinResp, err := client.RequestSignin(ctx, getFingerprint(ctx))
	if err != nil {
		return nil, jujuerrors.Annotate(err, "signing in")
	} else if signinResp.Data == nil {
		return nil, jujuerrors.New("no data received from control plane, please try again")
	}

	if config.G(ctx).LogType == log.TextType {
		log.G(ctx).Info().Msg(" ")
		log.G(ctx).Info().Msg("to authenticate, please visit:")
		log.G(ctx).Info().Msg(" ")
		log.G(ctx).Info().Msgf("  %s", *signinResp.Data.AuthorizationUrl)
		log.G(ctx).Info().Msg(" ")
	} else {
		log.G(ctx).
			Info().
			Str("url", *signinResp.Data.AuthorizationUrl).
			Msg("login")
	}

	checkResp, err := client.CheckAuthorization(ctx, controlplane.CheckAuthorizationRequest{
		RequestId: signinResp.Data.RequestId,
	})
	if err != nil {
		return nil, jujuerrors.Annotate(err, "checking authorization")
	}

	timeout := time.NewTimer(cmd.Timeout)
	ctx, cancel := context.WithCancel(ctx)

	var event *controlplane.Response[controlplane.CheckAuthorizationResponseData]
	go func() {
		defer cancel()
		for {
			select {
			case <-timeout.C:
				log.G(ctx).
					Error().
					Err(jujuerrors.New("login timed out, please try again"))
				return
			case event = <-checkResp:
				if event == nil {
					continue
				}
				return
			case <-ctx.Done():
				log.G(ctx).
					Error().
					Err(jujuerrors.Errorf("operation cancelled"))
				return
			}
		}
	}()

	if !cmd.NoBrowser {
		if err := browser.OpenURL(*signinResp.Data.AuthorizationUrl); err != nil {
			log.G(ctx).
				Debug().
				Err(err).
				Msg("could not open browser, please visit the URL manually")
		}
	}

	// TODO: run a spinner here
	log.G(ctx).
		Info().
		Str("timeout", cmd.Timeout.String()).
		Msg("waiting for confirmation")
	<-ctx.Done()

	if event == nil {
		return nil, jujuerrors.New("no event received, please try again")
	}

	return event, nil
}
