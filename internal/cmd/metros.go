// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/resource"
	"unikraft.com/cli/internal/resource/cmd"
)

type MetrosCmd struct {
	cmd.ResourceCmd[Metro]
	cmd.GettableResourceCmd[Metro] `set:"name=metro" set:"names=metros"`
	cmd.ListableResourceCmd[Metro] `set:"name=metro" set:"names=metros"`
}

type Metro struct {
	Name     string `field:",short" json:"name"`
	Country  string `field:",short" json:"country"`
	Endpoint string `field:",short" json:"endpoint"`
}

func (Metro) Type() resource.Type {
	return resource.Type{
		Name:  "metro",
		Names: "metros",
	}
}

func (i Metro) Key() resource.Key {
	return staticKey(i.Name)
}

func (i Metro) Raw() any {
	return i
}

func (i Metro) Fields() ([]resource.Field, error) {
	return resource.FieldsFromStruct(i)
}

func (Metro) List(ctx context.Context) ([]resource.Resource, error) {
	cfg := config.FromContextOrDefault(ctx)
	profile, err := cfg.CurrentProfile()
	if err != nil {
		return nil, err
	}

	var results []resource.Resource
	for _, metro := range profile.Metros {
		result := Metro{
			Name:     metro.Name,
			Country:  metro.Country,
			Endpoint: metro.Endpoint,
		}
		results = append(results, result)
	}
	return results, nil
}

func (Metro) Get(ctx context.Context, keys []string) ([]resource.Resource, error) {
	return getFromListable(ctx, Metro{}, keys)
}
