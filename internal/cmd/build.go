// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"

	"unikraft.com/cli/internal/builder"
	"unikraft.com/cli/internal/buildkit"
	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/images"
	imagespec "unikraft.com/x/image-spec"
	"unikraft.com/x/kraftfile"
)

type BuildCmd struct {
	Input  string `arg:"" default:"." help:"Path to the input directory."`
	Output string `short:"o" help:"Output destination"`

	// similar to docker compose build
	BuildArg []string `help:"Set build-time variables."`
	NoCache  bool     `help:"Do not use cache when building the image."`
	Secret   []string `help:"Secret to expose to the build (format: \"id=mysecret[,src=/local/secret]\")."`
	SSH      []string `help:"SSH agent socket or keys to expose to the build (format: \"default|<id>[=<socket>|<key>[,<key>]]\")."`
}

func (c *BuildCmd) Run(ctx context.Context, cfg *config.Config) error {
	kraftfile, err := kraftfile.ParseDirectory(c.Input)
	if err != nil {
		return err
	}

	buildOpts, err := builder.KraftfileToBuildOpts(c.Input, kraftfile)
	if err != nil {
		return err
	}

	if len(c.BuildArg) > 0 {
		buildOpts.Rootfs.BuildArg = append(buildOpts.Rootfs.BuildArg, c.BuildArg...)
	}
	if c.NoCache {
		buildOpts.Rootfs.NoCache = true
	}
	if len(c.Secret) > 0 {
		secrets, err := builder.ParseSecretSpecs(c.Secret)
		if err != nil {
			return err
		}
		buildOpts.Rootfs.Secrets = secrets
	}
	if len(c.SSH) > 0 {
		ssh, err := builder.ParseSSHSpecs(c.SSH)
		if err != nil {
			return err
		}
		buildOpts.Rootfs.SSH = ssh
	}

	bkc, cleanup, err := buildkit.ConnectToBuildkit(ctx)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	imgs, err := builder.Build(ctx, buildOpts, bkc)
	if err != nil {
		return err
	}
	defer func() {
		for _, img := range imgs {
			img.Close()
		}
	}()

	if c.Output == "" {
		return nil
	}
	output, err := imagespec.GuessURI(c.Output)
	if err != nil {
		return err
	}

	access, err := images.Accessor(ctx)
	if err != nil {
		return err
	}
	err = access.Save(ctx, output, imgs...)
	if err != nil {
		return err
	}

	return nil
}
