// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package instances

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"unikraft.com/cli/internal/config"

	"unikraft.com/cloud/sdk/platform"
)

type InstancesCmd struct{}

func (cmd *InstancesCmd) Help() string {
	return heredoc.Doc(`
		...
	`)
}

func (cmd *InstancesCmd) Run(cfg *config.Config) error {
	ctx := cfg.Context

	profile, err := cfg.CurrentProfile()
	if err != nil {
		return err
	}

	client := platform.NewClient(
		platform.WithToken(profile.Token),
		// HACK: default to Frankfurt for now
		platform.WithDefaultMetro("https://api.fra.unikraft.cloud"),
	)

	resp, err := client.GetInstances(ctx, nil, true)
	if err != nil {
		return err
	}
	for _, instance := range resp.Data.Instances {
		fmt.Println(*instance.Name, *instance.State)
	}

	return nil
}
