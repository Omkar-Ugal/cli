// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package login

import (
	"cmp"
	"context"
	"os"
	"strings"
	"time"

	jujuerrors "github.com/juju/errors"
	"github.com/pkg/browser"
	"unikraft.com/cloud/sdk/controlplane"
	"unikraft.com/x/log"
	"unikraft.com/x/ptr"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/httpclient"
	"unikraft.com/cli/internal/logfmt"
)

type LoginCmd struct {
	Check   bool          `name:"check" help:"Check if the user is already logged in."`
	Timeout time.Duration `short:"t" name:"timeout" default:"5m" help:"Timeout for the login request."`

	ControlPlane  string `name:"controlplane" default:"https://controlplane.unikraft.cloud" help:"Control plane URL to use for login."`
	AllowInsecure bool   `name:"allow-insecure" short:"k" help:"Allow insecure server connections when using SSL."`

	NoBrowser bool `name:"no-browser" help:"Do not open the browser automatically for login."`

	Token        *os.File `help:"Path to a file containing the authentication token (or '-' for stdin)."`
	Organization string   `help:"Organization to associate the login with."`
}

func (cmd *LoginCmd) Run(ctx context.Context, cfg *config.Config) error {
	if cmd.Token != nil {
		defer cmd.Token.Close()
	}

	profile, err := cfg.CurrentProfile()
	if err != nil {
		var notFound config.ErrProfileNotFound
		if !jujuerrors.Is(err, config.ErrNotSetup) && !jujuerrors.As(err, &notFound) {
			return jujuerrors.Annotate(err, "getting current profile")
		}

		profile = &config.Profile{
			Type: config.ProfileTypeCloud,
			Name: cfg.CurrentProfileName(),
		}
	}
	profile.ControlPlane = cmp.Or(cmd.ControlPlane, profile.ControlPlane, controlplane.DefaultEndpoint)

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
	profile.Token = ""
	profile.Organization = ""

	if cmd.Token != nil {
		log.G(ctx).Info().
			Msg("reading authentication token from file")

		dt, err := os.ReadFile(cmd.Token.Name())
		if err != nil {
			return jujuerrors.Annotate(err, "reading token file")
		}
		profile.Token = strings.TrimSpace(string(dt))
	} else {
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
	}
	profile.Organization = cmp.Or(profile.Organization, cmd.Organization)

	newMetros, err := cmd.getMetros(ctx, profile)
	if err != nil || len(newMetros) == 0 {
		log.G(ctx).
			Warn().
			Err(err).
			Msg("could not list metros for profile: please add metros manually")
	}
	existingMetros := make(map[string]struct{}, len(profile.Metros))
	for _, metro := range profile.Metros {
		existingMetros[metro.Name] = struct{}{}
	}
	for _, metro := range newMetros {
		if _, ok := existingMetros[metro.Name]; !ok {
			profile.Metros = append(profile.Metros, metro)
		}
	}

	cfg.DefaultProfile = profile.Name
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

func (cmd *LoginCmd) getMetros(ctx context.Context, profile *config.Profile) ([]config.Metro, error) {
	copts := []controlplane.ClientOption{
		controlplane.WithDefaultEndpoint(profile.ControlPlane),
		controlplane.WithToken(profile.Token),
	}
	if cmd.AllowInsecure {
		copts = append(copts, controlplane.WithHTTPClient(httpclient.InsecureHTTPClient))
	} else {
		copts = append(copts, controlplane.WithHTTPClient(httpclient.DefaultHTTPClient))
	}
	client := controlplane.NewClient(copts...)

	metroResp, err := client.ListMetros(ctx)
	if err != nil {
		return nil, err
	}
	if metroResp == nil || metroResp.Data == nil {
		return nil, nil
	}

	var metros []config.Metro
	for _, metro := range metroResp.Data.Metros {
		metros = append(metros, config.Metro{
			Name:     ptr.ZeroIfNil(metro.Name),
			Endpoint: ptr.ZeroIfNil(metro.Endpoint),
			Country:  ptr.ZeroIfNil(metro.Country),
		})
	}
	return metros, nil
}

func (cmd *LoginCmd) getAuth(ctx context.Context, profile *config.Profile) (*controlplane.Response[controlplane.CheckAuthorizationResponseData], error) {
	copts := []controlplane.ClientOption{
		controlplane.WithDefaultEndpoint(profile.ControlPlane),
	}
	if cmd.AllowInsecure {
		copts = append(copts, controlplane.WithHTTPClient(httpclient.InsecureHTTPClient))
	} else {
		copts = append(copts, controlplane.WithHTTPClient(httpclient.DefaultHTTPClient))
	}
	client := controlplane.NewClient(copts...)

	req, err := getFingerprint()
	if err != nil {
		return nil, jujuerrors.Annotate(err, "getting fingerprint")
	}
	signinResp, err := client.RequestSignin(ctx, req)
	if err != nil {
		return nil, jujuerrors.Annotate(err, "signing in")
	} else if signinResp.Data == nil {
		return nil, jujuerrors.New("no data received from control plane, please try again")
	}

	if logfmt.LogType(ctx) == log.TextType {
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
