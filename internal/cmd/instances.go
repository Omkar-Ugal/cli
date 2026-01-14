// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"unikraft.com/cloud/sdk/platform"
	"unikraft.com/x/log"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/logs"
	"unikraft.com/cli/internal/mirror"
	"unikraft.com/cli/internal/multimetro"
	"unikraft.com/cli/internal/resource"
	"unikraft.com/cli/internal/resource/cmd"
	xslices "unikraft.com/cli/internal/x/slices"
)

type InstancesCmd struct {
	cmd.ResourceCmd[Instance]          `set:"name=instance" set:"names=instances"`
	cmd.DeletableResourceCmd[Instance] `set:"name=instance" set:"names=instances"`
	cmd.EditableResourceCmd[Instance]  `set:"name=instance" set:"names=instances"`

	Logs    InstancesLogsCmd    `cmd:"" help:"Fetch and display instance logs"`
	Start   InstancesStartCmd   `cmd:"" help:"Start one or more instances"`
	Stop    InstancesStopCmd    `cmd:"" help:"Stop one or more instances"`
	Restart InstancesRestartCmd `cmd:"" help:"Restart one or more instances"`
}

type Instance struct {
	MetroName string `mirror:"metro.name" field:"metro,short"`
	Name      string `mirror:"instance.name" field:",short"`
	UUID      string `mirror:"instance.uuid" field:",long"`

	Tags []string `mirror:"instance.tags"`

	State InstanceState `mirror:"instance.state" field:",short"`
	Image string        `mirror:"instance.image" field:",short"`

	Runtime struct {
		Args []string          `mirror:"instance.args" field:",short"`
		Env  map[string]string `mirror:"instance.env" field:",long"`
	}

	Resources struct {
		Memory int `mirror:"instance.memory_mb" field:",short"`
		VCPUs  int `mirror:"instance.vcpus" field:"vcpus,short"`
	}

	Volumes []*struct {
		UUID     string `mirror:"uuid" field:",long"`
		Name     string `mirror:"name" field:",long"`
		At       string `mirror:"at" field:",long"`
		Readonly bool   `mirror:"readonly" field:",long"`
	} `mirror:"instance.volumes"`

	Service struct {
		UUID    string `mirror:"uuid" field:",long"`
		Name    string `mirror:"name" field:",long"`
		Domains []struct {
			FQDN string `mirror:"fqdn" field:",short"`
			// TODO: certificate
		} `mirror:"domains"`
	} `mirror:"instance.service_group"`

	Networks []struct {
		UUID      string `mirror:"uuid" field:",long"`
		PrivateIP string `mirror:"private_ip" field:",long"`
		MAC       string `mirror:"mac" field:",long"`
	} `mirror:"instance.network_interfaces"`

	Timestamps struct {
		CreatedAt time.Time `mirror:"instance.created_at"`
		StartedAt time.Time `mirror:"instance.started_at"`
		StoppedAt time.Time `mirror:"instance.stopped_at"`
	}

	ScaleToZero struct {
		Enabled      bool   `mirror:"instance.scale_to_zero.enabled"`
		Policy       string `mirror:"instance.scale_to_zero.policy"`
		Stateful     bool   `mirror:"instance.scale_to_zero.stateful"`
		CooldownTime int64  `mirror:"instance.scale_to_zero.cooldown_time_ms"`
	}

	Timing struct {
		Uptime   DurationMS `mirror:"instance.uptime_ms"`
		BootTime DurationUS `mirror:"instance.boot_time_us"`
		NetTime  DurationUS `mirror:"instance.net_time_us"`
	}

	Restart struct {
		Policy       string `mirror:"instance.restart_policy"`
		StartCount   int    `mirror:"instance.start_count"`
		RestartCount int    `mirror:"instance.restart_count"`
	}

	Stop struct {
		Reason   int `mirror:"instance.stop_reason"`
		ExitCode int `mirror:"instance.exit_code"`
		Code     int `mirror:"instance.stop_code"`
	}

	Instance platform.Instance `field:"-" json:"instance"`
	Metro    *config.Metro     `field:"-" json:"metro"`

	organization string
}

func (Instance) Type() resource.Type {
	return resource.Type{
		Name:  "instance",
		Names: "instances",
	}
}

func (i Instance) key() multimetro.Key {
	return multimetro.Key{
		Metro: i.Metro.Name,
		Name:  i.Name,
		UUID:  i.UUID,
	}
}

func (i Instance) Key() string {
	return i.key().String()
}

func (i Instance) Raw() any {
	return i.Instance
}

func (i Instance) Fields() ([]resource.Field, error) {
	result, err := resource.FieldsFromStruct(i)
	if err != nil {
		return nil, err
	}

	for key, field := range resource.IterFields(result) {
		switch key.String() {
		case "name":
			field.Hyperlink = i.hyperlink()
		case "image":
			field.Patch = &resource.Patch{Set: ""}
		case "runtime.args":
			field.Patch = &resource.Patch{Set: []string{}}
		case "runtime.env":
			field.Patch = &resource.Patch{
				Set: map[string]string{},
				Add: map[string]string{},
				Del: []string{},
			}
		case "resources.memory":
			field.Patch = &resource.Patch{Set: 0}
		case "resources.vcpus":
			field.Patch = &resource.Patch{Set: 0}
		}
	}

	return result, nil
}

func (i Instance) hyperlink() string {
	if i.organization == "" || i.Name == "" {
		return ""
	}
	metro := i.Metro.Name
	return fmt.Sprintf("https://console.unikraft.cloud/org/%s/instances/%s/%s", i.organization, metro, i.Name)
}

func (Instance) List(ctx context.Context) ([]resource.Resource, error) {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	resources, err := multimetro.DoAll(ctx, cl, func(ctx context.Context, mc *multimetro.MetroClient) ([]resource.Resource, error) {
		log.G(ctx).Trace().Msg("listing instances")
		resp, err := mc.GetInstances(ctx, nil, true)
		if err != nil {
			return nil, err
		}
		var results []resource.Resource
		for _, instance := range resp.Data.Instances {
			result, err := Instance{}.load(ctx, instance, &mc.Metro)
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

func (Instance) Get(ctx context.Context, keys []string) ([]resource.Resource, error) {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	resources, err := multimetro.DoKeys(ctx, cl, multimetro.ParseKeys(keys), func(ctx context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]resource.Resource, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("getting instances")
		resp, err := mc.GetInstances(ctx, keys.NamesOrUUIDs(), true)
		if err != nil && !platform.ErrorContainsOnly(err, platform.APIHTTPErrorNotFound) {
			return nil, nil, err
		}
		var found []multimetro.Key
		var results []resource.Resource
		for _, instance := range resp.Data.Instances {
			if instance.Status == nil || *instance.Status != platform.ResponseStatusSUCCESS {
				continue
			}
			result, err := Instance{}.load(ctx, instance, &mc.Metro)
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

func (Instance) load(ctx context.Context, instance platform.Instance, metro *config.Metro) (Instance, error) {
	result := Instance{
		Instance: instance,
		Metro:    metro,
	}
	err := mirror.Mirror(result, &result)
	if err != nil {
		return Instance{}, fmt.Errorf("could not mirror instance data: %w", err)
	}

	if name, _, ok := strings.Cut(result.Image, "@"); ok {
		result.Image = name
	}

	cfg := config.FromContextOrDefault(ctx)
	profile, err := cfg.CurrentProfile()
	if err != nil {
		return Instance{}, err
	}
	result.organization = profile.Organization

	return result, nil
}

func (Instance) Delete(ctx context.Context, targets []resource.Resource) error {
	keys := make(multimetro.Keys, 0, len(targets))
	for _, target := range targets {
		instance := target.(Instance)
		keys = append(keys, instance.key())
	}

	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return err
	}
	_, err = multimetro.DoKeys(ctx, cl, keys, func(ctx context.Context, metroClient *multimetro.MetroClient, keys multimetro.Keys) ([]struct{}, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("deleting instances")
		instances, err := metroClient.DeleteInstances(ctx, keys.NamesOrUUIDs())
		if err != nil {
			return nil, nil, err
		}
		var deleted []multimetro.Key
		for _, instance := range instances.Data.Instances {
			if instance.Status == nil || *instance.Status != platform.ResponseStatusSUCCESS {
				continue
			}
			deleted = append(deleted, multimetro.Key{
				Metro: metroClient.Metro.Name,
				Name:  *instance.Name,
				UUID:  *instance.Uuid,
			})
		}
		return nil, deleted, nil
	})
	return err
}

func (Instance) Edit(ctx context.Context, target resource.Resource, fields []resource.Field) (resource.Resource, error) {
	instance := target.(Instance)
	var reqs []platform.UpdateInstancesRequestItem
	for key, field := range resource.IterFields(fields) {
		reqs = append(reqs, Instance{}.getFieldRequests(instance.UUID, key, *field)...)
	}

	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	_, err = multimetro.DoKeyExact(ctx, cl, instance.key(), func(ctx context.Context, metroClient *multimetro.MetroClient) (struct{}, error) {
		log.G(ctx).Trace().Msg("updating instance")
		_, err := metroClient.UpdateInstances(ctx, reqs)
		return struct{}{}, err
	})
	if err != nil {
		return nil, err
	}
	results, err := Instance{}.Get(ctx, []string{instance.Key()})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (Instance) getFieldRequests(uuid string, path resource.FieldPath, field resource.Field) (reqs []platform.UpdateInstancesRequestItem) {
	if field.Patch == nil {
		return reqs
	}
	if field.Patch.Set != nil {
		reqs = append(reqs, Instance{}.getPatchRequest(uuid, path, field.Patch.Set, platform.UpdateInstancesRequestItemOpSet))
	}
	if field.Patch.Add != nil {
		reqs = append(reqs, Instance{}.getPatchRequest(uuid, path, field.Patch.Add, platform.UpdateInstancesRequestItemOpAdd))
	}
	if field.Patch.Del != nil {
		reqs = append(reqs, Instance{}.getPatchRequest(uuid, path, field.Patch.Del, platform.UpdateInstancesRequestItemOpDel))
	}
	return reqs
}

func (Instance) getPatchRequest(uuid string, key resource.FieldPath, value any, op platform.UpdateInstancesRequestItemOp) platform.UpdateInstancesRequestItem {
	var prop platform.UpdateInstancesRequestItemProp
	switch key.String() {
	case "image":
		prop = platform.UpdateInstancesRequestItemPropImage
	case "runtime.args":
		prop = platform.UpdateInstancesRequestItemPropArgs
	case "runtime.env":
		prop = platform.UpdateInstancesRequestItemPropEnv
	case "resources.memory":
		prop = platform.UpdateInstancesRequestItemPropMemory_mb
	case "resources.vcpus":
		prop = platform.UpdateInstancesRequestItemPropVcpus
	default:
		return platform.UpdateInstancesRequestItem{}
	}
	return platform.UpdateInstancesRequestItem{
		Uuid:  &uuid,
		Op:    op,
		Prop:  prop,
		Value: platform.Ptr(value),
	}
}

type InstancesLogsCmd struct {
	Name []string `arg:"" help:"Names of the instances to fetch logs for."`

	Tail   int  `help:"Number of lines to show from the end of the logs."`
	Follow bool `short:"f" help:"Follow log output."`
}

func (cmd *InstancesLogsCmd) Run(ctx context.Context, cfg *config.Config) error {
	// HACK: we resolve the keys early, so that we can assume that all the
	// instances actually exist (this is a potential race condition, but it's
	// acceptable for now)
	instances, err := Instance{}.Get(ctx, cmd.Name)
	if err != nil {
		return err
	}
	keys := make(multimetro.Keys, 0, len(instances))
	for _, instance := range instances {
		key := instance.(Instance).key()
		if key.Metro == "" {
			return fmt.Errorf("key %q not fully resolved", key)
		}
		keys = append(keys, key)
	}

	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	_, err = multimetro.DoKeys(ctx, cl, keys, func(_ context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]struct{}, []multimetro.Key, error) {
		for _, key := range keys {
			eg.Go(func() error {
				r, err := logs.InstanceLogs(ctx, mc).Reader(key.NameOrUUID(), cmd.Tail, cmd.Follow)
				if err != nil {
					return err
				}
				_, err = io.Copy(cfg.Stdout, r)
				return err
			})
		}
		return nil, keys, nil
	})
	if err != nil {
		return err
	}
	err = eg.Wait()
	if cmd.Follow && errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

type InstancesStartCmd struct {
	Name []string `arg:"" help:"Names of the instances to start."`
}

func (cmd *InstancesStartCmd) Run(ctx context.Context) error {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return err
	}
	_, err = multimetro.DoKeys(ctx, cl, multimetro.ParseKeys(cmd.Name), func(ctx context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]struct{}, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("starting instances")
		resp, err := mc.StartInstances(ctx, keys.NamesOrUUIDs())
		if err != nil && !platform.ErrorContainsOnly(err, platform.APIHTTPErrorNotFound) {
			return nil, nil, err
		}
		var started []multimetro.Key
		for _, instance := range resp.Data.Instances {
			if instance.Status == nil || *instance.Status != platform.ResponseStatusSUCCESS {
				continue
			}
			started = append(started, multimetro.Key{
				Metro: mc.Metro.Name,
				Name:  *instance.Name,
				UUID:  *instance.Uuid,
			})
		}
		return nil, started, nil
	})
	// FIXME: show diff view
	return err
}

type InstancesStopCmd struct {
	Name []string `arg:"" help:"Names of the instances to stop."`

	StopOpts
}

func (cmd *InstancesStopCmd) Run(ctx context.Context) error {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return err
	}
	_, err = multimetro.DoKeys(ctx, cl, multimetro.ParseKeys(cmd.Name), func(ctx context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]struct{}, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("stopping instances")
		reqs := make([]platform.StopInstancesRequestItem, 0, len(keys))
		for _, key := range keys {
			reqs = append(reqs, cmd.toReq(key.NameOrUUID()))
		}
		resp, err := mc.StopInstances(ctx, reqs)
		if err != nil && !platform.ErrorContainsOnly(err, platform.APIHTTPErrorNotFound) {
			return nil, nil, err
		}
		var stopped []multimetro.Key
		for _, instance := range resp.Data.Instances {
			if instance.Status == nil || *instance.Status != platform.ResponseStatusSUCCESS {
				continue
			}
			stopped = append(stopped, multimetro.Key{
				Metro: mc.Metro.Name,
				Name:  *instance.Name,
				UUID:  *instance.Uuid,
			})
		}
		return nil, stopped, nil
	})
	// FIXME: show diff view
	return err
}

type InstancesRestartCmd struct {
	Name []string `arg:"" help:"Names of the instances to restart."`

	StopOpts
}

func (cmd *InstancesRestartCmd) Run(ctx context.Context) error {
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return err
	}

	// First stop the instances
	keys := multimetro.ParseKeys(cmd.Name)
	keys, err = multimetro.DoKeys(ctx, cl, keys, func(ctx context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]multimetro.Key, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("stopping instances for restart")
		reqs := make([]platform.StopInstancesRequestItem, 0, len(keys))
		for _, key := range keys {
			reqs = append(reqs, cmd.toReq(key.NameOrUUID()))
		}
		resp, err := mc.StopInstances(ctx, reqs)
		if err != nil && !platform.ErrorContainsOnly(err, platform.APIHTTPErrorNotFound) {
			return nil, nil, err
		}
		var stopped []multimetro.Key
		for _, instance := range resp.Data.Instances {
			if instance.Status == nil || *instance.Status != platform.ResponseStatusSUCCESS {
				continue
			}
			stopped = append(stopped, multimetro.Key{
				Metro: mc.Metro.Name,
				Name:  *instance.Name,
				UUID:  *instance.Uuid,
			})
		}
		return stopped, stopped, nil
	})
	if err != nil {
		return err
	}

	// Then start the instances
	_, err = multimetro.DoKeys(ctx, cl, keys, func(ctx context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]struct{}, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("starting instances for restart")
		resp, err := mc.StartInstances(ctx, keys.NamesOrUUIDs())
		if err != nil && !platform.ErrorContainsOnly(err, platform.APIHTTPErrorNotFound) {
			return nil, nil, err
		}
		var started []multimetro.Key
		for _, instance := range resp.Data.Instances {
			if instance.Status == nil || *instance.Status != platform.ResponseStatusSUCCESS {
				continue
			}
			started = append(started, multimetro.Key{
				Metro: mc.Metro.Name,
				Name:  *instance.Name,
				UUID:  *instance.Uuid,
			})
		}
		return nil, started, nil
	})

	// FIXME: show diff view (start count will be different)

	return err
}

type StopOpts struct {
	Force        bool  `help:"Force stop the instance immediately."`
	DrainTimeout int64 `help:"Timeout in milliseconds for draining connections before stopping." default:"-1"`
}

func (args *StopOpts) toReq(nameOrUUID platform.NameOrUUID) platform.StopInstancesRequestItem {
	req := platform.StopInstancesRequestItem{
		Uuid: nameOrUUID.Uuid,
		Name: nameOrUUID.Name,
	}
	if args.Force {
		req.Force = &args.Force
	}
	if args.DrainTimeout >= 0 {
		timeout := uint64(args.DrainTimeout)
		req.DrainTimeoutMs = &timeout
	}
	return req
}
