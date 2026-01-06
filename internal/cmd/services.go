// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"
	"fmt"
	"time"

	"unikraft.com/cloud/sdk/platform"
	"unikraft.com/x/log"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/mirror"
	"unikraft.com/cli/internal/multimetro"
	"unikraft.com/cli/internal/resource"
	xslices "unikraft.com/cli/internal/x/slices"
)

type ServicesCmd struct {
	resource.ResourceCmd[ServiceGroup] `set:"name=service" set:"names=services"`
}

type ServiceGroup struct {
	MetroName string `mirror:"metro.name" field:"metro,short"`
	Name      string `mirror:"service_group.name" field:",short"`
	UUID      string `mirror:"service_group.uuid" field:",long"`

	Persistent bool `mirror:"service_group.persistent" field:",long"`
	Autoscale  bool `mirror:"service_group.autoscale" field:",short"`

	Limits struct {
		Soft uint64 `mirror:"service_group.soft_limit" field:",long"`
		Hard uint64 `mirror:"service_group.hard_limit" field:",long"`
	}

	Timestamps struct {
		CreatedAt time.Time `mirror:"service_group.created_at"`
	}

	Domains []struct {
		FQDN string `mirror:"fqdn" field:",short"`
		// TODO: certificate
	} `mirror:"service_group.domains"`

	Instances []struct {
		Name string `mirror:"name" field:",long"`
		UUID string `mirror:"uuid" field:",long"`
	} `mirror:"service_group.instances"`

	// TODO: support shorthand
	Services []struct {
		Source      uint32                     `mirror:"port" field:",long"`
		Destination uint32                     `mirror:"destination_port" field:",long"`
		Handlers    []platform.ServiceHandlers `mirror:"handlers" field:",long"`
	} `mirror:"service_group.services"`

	ServiceGroup platform.ServiceGroup `field:"-" json:"service_group"`
	Metro        *config.Metro         `field:"-" json:"metro"`
}

func (ServiceGroup) Type() resource.Type {
	return resource.Type{
		Name:  "service",
		Names: "services",
	}
}

func (s ServiceGroup) key() multimetro.Key {
	return multimetro.Key{
		Metro: s.Metro.Name,
		Name:  s.Name,
		UUID:  s.UUID,
	}
}

func (s ServiceGroup) Key() string {
	return s.key().String()
}

func (s ServiceGroup) Raw() any {
	return s.ServiceGroup
}

func (s ServiceGroup) Fields() ([]resource.Field, error) {
	result, err := resource.FieldsFromStruct(s)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (ServiceGroup) List(ctx context.Context) ([]resource.Resource, error) {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	resources, err := multimetro.DoAll(ctx, cl, func(ctx context.Context, mc *multimetro.MetroClient) ([]resource.Resource, error) {
		log.G(ctx).Trace().Msg("listing service groups")
		resp, err := mc.GetServiceGroups(ctx, nil, true)
		if err != nil {
			return nil, err
		}
		var results []resource.Resource
		for _, serviceGroup := range resp.Data.ServiceGroups {
			result, err := ServiceGroup{}.load(serviceGroup, &mc.Metro)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return results, nil
	})
	if err != nil {
		return nil, err
	}
	return xslices.Flatten(resources), nil
}

func (ServiceGroup) Get(ctx context.Context, keys []string) ([]resource.Resource, error) {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	resources, err := multimetro.DoKeys(ctx, cl, multimetro.ParseKeys(keys), func(ctx context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]resource.Resource, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("getting service groups")
		resp, err := mc.GetServiceGroups(ctx, keys.NamesOrUUIDs(), true)
		if err != nil && !platform.ErrorContainsOnly(err, platform.APIHTTPErrorNotFound) {
			return nil, nil, err
		}
		var found []multimetro.Key
		var results []resource.Resource
		for _, serviceGroup := range resp.Data.ServiceGroups {
			if serviceGroup.Status == nil || *serviceGroup.Status != platform.ResponseStatusSUCCESS {
				continue
			}
			result, err := ServiceGroup{}.load(serviceGroup, &mc.Metro)
			if err != nil {
				return nil, nil, err
			}
			found = append(found, multimetro.Key{
				Metro: mc.Metro.Name,
				Name:  result.Name,
				UUID:  result.UUID,
			})
			results = append(results, result)
		}
		return results, found, nil
	})
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func (ServiceGroup) load(serviceGroup platform.ServiceGroup, metro *config.Metro) (ServiceGroup, error) {
	result := ServiceGroup{
		ServiceGroup: serviceGroup,
		Metro:        metro,
	}
	err := mirror.Mirror(result, &result)
	if err != nil {
		return ServiceGroup{}, fmt.Errorf("could not mirror service group data: %w", err)
	}
	return result, nil
}
