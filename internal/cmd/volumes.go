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

type VolumesCmd struct {
	resource.ResourceCmd[Volume]          `set:"name=volume" set:"names=volumes"`
	resource.DeletableResourceCmd[Volume] `set:"name=volume" set:"names=volumes"`
	resource.EditableResourceCmd[Volume]  `set:"name=volume" set:"names=volumes"`
	resource.CreatableResourceCmd[Volume] `set:"name=volume" set:"names=volumes"`
}

type Volume struct {
	MetroName string `mirror:"metro.name" field:"metro,short"`
	Name      string `mirror:"volume.name" field:",short"`
	UUID      string `mirror:"volume.uuid" field:",long"`

	Tags []string `mirror:"volume.tags"`

	State      VolumeState `mirror:"volume.state" field:",short"`
	Size       SizeMB      `mirror:"volume.size_mb" field:",short"`
	Persistent bool        `mirror:"volume.persistent" field:",long"`

	Timestamps struct {
		CreatedAt time.Time `mirror:"volume.created_at"`
	}

	AttachedTo []struct {
		Name string `mirror:"name" field:",long"`
		UUID string `mirror:"uuid" field:",long"`
	} `mirror:"volume.attached_to"`

	MountedBy []struct {
		Name     string `mirror:"name" field:",long"`
		UUID     string `mirror:"uuid" field:",long"`
		ReadOnly bool   `mirror:"read_only" field:",long"`
	} `mirror:"volume.mounted_by"`

	Volume platform.Volume `field:"-" json:"volume"`
	Metro  *config.Metro   `field:"-" json:"metro"`
}

func (Volume) Type() resource.Type {
	return resource.Type{
		Name:  "volume",
		Names: "volumes",
	}
}

func (i Volume) key() multimetro.Key {
	return multimetro.Key{
		Metro: i.Metro.Name,
		Name:  i.Name,
		UUID:  i.UUID,
	}
}

func (i Volume) Key() string {
	return i.key().String()
}

func (i Volume) Raw() any {
	return i.Volume
}

func (i Volume) Fields() ([]resource.Field, error) {
	result, err := resource.FieldsFromStruct(i)
	if err != nil {
		return nil, err
	}

	for idx, field := range result {
		switch field.Name {
		case "name":
			result[idx].Create = &resource.Patch{
				Set: "",
			}
		case "metro":
			result[idx].Create = &resource.Patch{
				Set:      "",
				Required: true,
			}
		case "size":
			result[idx].Create = &resource.Patch{
				Set:      SizeMB(0),
				Required: true,
			}
			result[idx].Patch = &resource.Patch{
				Set: SizeMB(0),
			}
		}
	}

	return result, nil
}

func (Volume) List(ctx context.Context) ([]resource.Resource, error) {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	resources, err := multimetro.DoAll(ctx, cl, func(ctx context.Context, mc *multimetro.MetroClient) ([]resource.Resource, error) {
		log.G(ctx).Trace().Msg("listing volumes")
		resp, err := mc.GetVolumes(ctx, nil, true)
		if err != nil {
			return nil, err
		}
		var results []resource.Resource
		for _, volume := range resp.Data.Volumes {
			result, err := Volume{}.load(volume, &mc.Metro)
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

func (Volume) Get(ctx context.Context, keys []string) ([]resource.Resource, error) {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	resources, err := multimetro.DoKeys(ctx, cl, multimetro.ParseKeys(keys), func(ctx context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]resource.Resource, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("getting volumes")
		resp, err := mc.GetVolumes(ctx, keys.NamesOrUUIDs(), true)
		if err != nil && !platform.ErrorContainsOnly(err, platform.APIHTTPErrorNotFound) {
			return nil, nil, err
		}
		var found []multimetro.Key
		var results []resource.Resource
		for _, volume := range resp.Data.Volumes {
			if volume.Status == nil || *volume.Status != "success" {
				continue
			}
			result, err := Volume{}.load(volume, &mc.Metro)
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

func (Volume) load(volume platform.Volume, metro *config.Metro) (Volume, error) {
	result := Volume{
		Volume: volume,
		Metro:  metro,
	}
	err := mirror.Mirror(result, &result)
	if err != nil {
		return Volume{}, fmt.Errorf("could not mirror volume data: %w", err)
	}
	return result, nil
}

func (Volume) Delete(ctx context.Context, targets []resource.Resource) error {
	keys := make(multimetro.Keys, 0, len(targets))
	for _, target := range targets {
		volume := target.(Volume)
		keys = append(keys, volume.key())
	}

	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return err
	}
	_, err = multimetro.DoKeys(ctx, cl, keys, func(ctx context.Context, metroClient *multimetro.MetroClient, keys multimetro.Keys) ([]struct{}, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("deleting volumes")
		var deleted []multimetro.Key
		for _, key := range keys {
			_, err := metroClient.DeleteVolumeByUUID(ctx, key.UUID)
			if err != nil {
				return nil, nil, err
			}
			deleted = append(deleted, key)
		}
		return nil, deleted, nil
	})
	return err
}

func (Volume) Create(ctx context.Context, fields []resource.Field) (resource.Resource, error) {
	var req platform.CreateVolumeRequest
	var metro string
	for key, field := range resource.IterFields(fields) {
		if field.Create.Set != nil {
			switch key.String() {
			case "name":
				name := field.Create.Set.(string)
				req.Name = &name
			case "metro":
				metro = field.Create.Set.(string)
			case "size":
				size := field.Create.Set.(SizeMB)
				req.SizeMb = uint64(size)
			}
		}
	}

	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	uuid, err := multimetro.DoMetro(ctx, cl, metro, func(ctx context.Context, mc *multimetro.MetroClient) (string, error) {
		log.G(ctx).Trace().Msg("creating volume")
		resp, err := mc.CreateVolume(ctx, req)
		if err != nil {
			return "", err
		}
		return *resp.Data.Volumes[0].Uuid, nil
	})
	if err != nil {
		return nil, err
	}

	key := multimetro.Key{
		Metro: metro,
		UUID:  uuid,
	}
	results, err := Volume{}.Get(ctx, []string{key.String()})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (Volume) Edit(ctx context.Context, target resource.Resource, fields []resource.Field) (resource.Resource, error) {
	volume := target.(Volume)
	var reqs []platform.UpdateVolumesRequestItem
	for key, field := range resource.IterFields(fields) {
		reqs = append(reqs, Volume{}.getFieldRequests(volume.UUID, key, *field)...)
	}

	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	_, err = multimetro.DoKeyExact(ctx, cl, volume.key(), func(ctx context.Context, metroClient *multimetro.MetroClient) (struct{}, error) {
		log.G(ctx).Trace().Msg("updating volume")
		_, err := metroClient.UpdateVolumes(ctx, reqs)
		return struct{}{}, err
	})
	if err != nil {
		return nil, err
	}
	results, err := Volume{}.Get(ctx, []string{volume.Key()})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (Volume) getFieldRequests(uuid string, key resource.FieldPath, field resource.Field) (reqs []platform.UpdateVolumesRequestItem) {
	if field.Patch == nil {
		return reqs
	}
	if field.Patch.Set != nil {
		reqs = append(reqs, Volume{}.getPatchRequest(uuid, key, field.Patch.Set, platform.UpdateVolumesRequestItemOpSet))
	}
	if field.Patch.Add != nil {
		reqs = append(reqs, Volume{}.getPatchRequest(uuid, key, field.Patch.Add, platform.UpdateVolumesRequestItemOpAdd))
	}
	if field.Patch.Del != nil {
		reqs = append(reqs, Volume{}.getPatchRequest(uuid, key, field.Patch.Del, platform.UpdateVolumesRequestItemOpDel))
	}
	return reqs
}

func (Volume) getPatchRequest(uuid string, key resource.FieldPath, value any, op platform.UpdateVolumesRequestItemOp) platform.UpdateVolumesRequestItem {
	var prop platform.UpdateVolumesRequestItemProp
	switch key.String() {
	case "size":
		prop = platform.UpdateVolumesRequestItemPropSize_mb
	default:
		return platform.UpdateVolumesRequestItem{}
	}
	return platform.UpdateVolumesRequestItem{
		Uuid:  &uuid,
		Op:    op,
		Prop:  prop,
		Value: platform.Ptr(value),
	}
}
