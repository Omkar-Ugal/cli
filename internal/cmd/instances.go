// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"unikraft.com/cloud/sdk/platform"
	"unikraft.com/x/kingkong"
	"unikraft.com/x/log"
	"unikraft.com/x/ptr"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/logs"
	"unikraft.com/cli/internal/mirror"
	"unikraft.com/cli/internal/multimetro"
	"unikraft.com/cli/internal/resource"
	"unikraft.com/cli/internal/resource/cmd"
	"unikraft.com/cli/internal/types"
	xslices "unikraft.com/cli/internal/x/slices"
)

type InstancesCmd struct {
	cmd.ResourceCmd[Instance]
	cmd.GettableResourceCmd[Instance]  `set:"name=instance" set:"names=instances"`
	cmd.ListableResourceCmd[Instance]  `set:"name=instance" set:"names=instances"`
	cmd.DeletableResourceCmd[Instance] `set:"name=instance" set:"names=instances"`
	cmd.EditableResourceCmd[Instance]  `set:"name=instance" set:"names=instances"`
	cmd.CreatableResourceCmd[Instance] `set:"name=instance" set:"names=instances"`

	Logs    InstancesLogsCmd    `cmd:"" help:"Fetch and display instance logs"`
	Start   InstancesStartCmd   `cmd:"" help:"Start one or more instances"`
	Stop    InstancesStopCmd    `cmd:"" help:"Stop one or more instances"`
	Restart InstancesRestartCmd `cmd:"" help:"Restart one or more instances"`
}

type Instance struct {
	MetroName string `mirror:"metro.name" field:"metro,short" create:"set,required"`
	Name      string `mirror:"instance.name" field:",short" create:"set"`
	UUID      string `mirror:"instance.uuid" field:",long"`

	Tags []string `mirror:"instance.tags"`

	State types.InstanceState `mirror:"instance.state" field:",short"`
	Image string              `mirror:"instance.image" field:",short" create:"set,required" edit:"set"`

	Runtime struct {
		Args []string          `mirror:"instance.args" field:",short" create:"set" edit:"set"`
		Env  map[string]string `mirror:"instance.env" field:",long" create:"set" edit:"set,add,del=keys"`
	}

	Resources struct {
		Memory types.SizeMebibytes `mirror:"instance.memory_mb" field:",short" create:"set" edit:"set"`
		VCPUs  int                 `mirror:"instance.vcpus" field:"vcpus,short" create:"set" edit:"set"`
	}

	Volumes []*InstanceVolume `mirror:"instance.volumes" field:",embed" create:"set"`

	Service struct {
		UUID    string   `mirror:"uuid" field:",long" create:"set"`
		Name    string   `mirror:"name" field:",long" create:"set"`
		Domains []Domain `mirror:"domains" field:",short,embed" create:"set"`

		// create-only fields
		Services  []*Service `field:",invisible,embed" create:"set"`
		SoftLimit uint32     `field:",long" create:"set"`
		HardLimit uint32     `field:",long" create:"set"`
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

	ScaleToZero InstanceScaleToZero `field:",embed" mirror:"instance.scale_to_zero"`

	Timing struct {
		Uptime   types.DurationMS `mirror:"instance.uptime_ms"`
		BootTime types.DurationUS `mirror:"instance.boot_time_us"`
		NetTime  types.DurationUS `mirror:"instance.net_time_us"`
	}

	Restart struct {
		Policy       string `mirror:"instance.restart_policy" create:"set"`
		StartCount   int    `mirror:"instance.start_count"`
		RestartCount int    `mirror:"instance.restart_count"`
	}

	Stop struct {
		Reason   int `mirror:"instance.stop_reason"`
		ExitCode int `mirror:"instance.exit_code"`
		Code     int `mirror:"instance.stop_code"`
	}

	Autostart     bool     `field:",long" create:"set"`
	Replicas      int64    `field:",long" create:"set"`
	WaitTimeoutMs int64    `field:"wait_timeout_ms,long" create:"set"`
	Features      []string `field:",long" create:"set"`

	Instance platform.Instance `field:"-" json:"instance"`
	Metro    *config.Metro     `field:"-" json:"metro"`

	organization string
}

type InstanceVolume struct {
	UUID     string `mirror:"uuid" json:"uuid,omitempty" field:",long"`
	Name     string `mirror:"name" json:"name,omitempty" field:",long"`
	SizeMB   int64  `json:"size_mb,omitempty"`
	At       string `mirror:"at" json:"at" field:",long"`
	Readonly bool   `mirror:"readonly" json:"readonly,omitempty" field:",long"`
}

func (v *InstanceVolume) MarshalText() ([]byte, error) {
	parts := []string{cmp.Or(v.Name, v.UUID)}
	if v.SizeMB > 0 {
		parts = append(parts, strconv.FormatInt(v.SizeMB, 10)+"M")
	}
	parts = append(parts, v.At)
	if v.Readonly {
		parts = append(parts, "ro")
	}
	return []byte(strings.Join(parts, ":")), nil
}

func (v *InstanceVolume) UnmarshalText(data []byte) error {
	str := string(data)
	parts := strings.Split(str, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid volume format, expected NAME:AT[:ro] or UUID:AT[:ro] or NAME:SIZE:AT[:ro]")
	}

	// Check if last part is "ro"
	if len(parts) > 0 && parts[len(parts)-1] == "ro" {
		v.Readonly = true
		parts = parts[:len(parts)-1]
	}

	if len(parts) == 2 {
		// Could be NAME:AT or UUID:AT
		v.Name = parts[0]
		v.UUID = parts[0]
		v.At = parts[1]
	} else if len(parts) == 3 {
		// NAME:SIZE:AT
		v.Name = parts[0]
		sizeStr := strings.TrimSuffix(parts[1], "M")
		size, err := strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid size: %w", err)
		}
		v.SizeMB = size
		v.At = parts[2]
	} else {
		return fmt.Errorf("invalid volume format")
	}

	return nil
}

type InstanceScaleToZero struct {
	Enabled      bool   `mirror:"enabled" field:",long"`
	Policy       string `mirror:"policy" field:",long" create:"set"`
	Stateful     bool   `mirror:"stateful" field:",long" create:"set"`
	CooldownTime int64  `mirror:"cooldown_time_ms" field:",long" create:"set"`
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

func (i Instance) Key() resource.Key {
	return i.key()
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
		case "service":
			nameField, _ := field.Get("name")
			uuidField, _ := field.Get("uuid")
			name, _ := nameField.Value.(string)
			uuid, _ := uuidField.Value.(string)
			if name != "" || uuid != "" {
				field.Links = append(field.Links, resource.Link{
					Type: "service",
					Key: multimetro.Key{
						Metro: i.Metro.Name,
						Name:  name,
						UUID:  uuid,
					}.String(),
				})
			}
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
	results, err := Instance{}.Get(ctx, []string{instance.Key().String()})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (Instance) getFieldRequests(uuid string, path resource.FieldPath, field resource.Field) (reqs []platform.UpdateInstancesRequestItem) {
	if field.Edit == nil {
		return reqs
	}
	if field.Edit.Set != nil {
		reqs = append(reqs, Instance{}.getPatchRequest(uuid, path, field.Edit.Set, platform.UpdateInstancesRequestItemOpSet))
	}
	if field.Edit.Add != nil {
		reqs = append(reqs, Instance{}.getPatchRequest(uuid, path, field.Edit.Add, platform.UpdateInstancesRequestItemOpAdd))
	}
	if field.Edit.Del != nil {
		reqs = append(reqs, Instance{}.getPatchRequest(uuid, path, field.Edit.Del, platform.UpdateInstancesRequestItemOpDel))
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

func (Instance) Create(ctx context.Context, fields []resource.Field) ([]resource.Resource, error) {
	var req platform.CreateInstanceRequest
	var metro string
	for key, field := range resource.IterFields(fields) {
		if field.Create == nil || field.Create.Set == nil {
			continue
		}
		switch key.String() {
		case "name":
			name := field.Create.Set.(string)
			req.Name = &name
		case "metro":
			metro = field.Create.Set.(string)
		case "image":
			req.Image = field.Create.Set.(string)
		case "runtime.args":
			req.Args = field.Create.Set.([]string)
		case "runtime.env":
			req.Env = field.Create.Set.(map[string]string)
		case "resources.memory":
			mem := int64(field.Create.Set.(types.SizeMebibytes))
			req.MemoryMb = &mem
		case "resources.vcpus":
			vcpus := int32(field.Create.Set.(int))
			req.Vcpus = &vcpus
		case "restart.policy":
			policy := platform.CreateInstanceRequestRestartPolicy(field.Create.Set.(string))
			req.RestartPolicy = &policy
		case "scale-to-zero.policy":
			if req.ScaleToZero == nil {
				req.ScaleToZero = &platform.CreateInstanceRequestScaleToZero{}
			}
			policy := platform.CreateInstanceRequestScaleToZeroPolicy(field.Create.Set.(string))
			req.ScaleToZero.Policy = &policy
		case "scale-to-zero.stateful":
			if req.ScaleToZero == nil {
				req.ScaleToZero = &platform.CreateInstanceRequestScaleToZero{}
			}
			stateful := field.Create.Set.(bool)
			req.ScaleToZero.Stateful = &stateful
		case "scale-to-zero.cooldown-time":
			if req.ScaleToZero == nil {
				req.ScaleToZero = &platform.CreateInstanceRequestScaleToZero{}
			}
			cooldown := int32(field.Create.Set.(int64))
			req.ScaleToZero.CooldownTimeMs = &cooldown
		case "volumes":
			for _, vol := range field.Create.Set.([]*InstanceVolume) {
				reqVol := platform.CreateInstanceRequestVolume{
					At: vol.At,
				}
				if vol.UUID != "" {
					reqVol.Uuid = &vol.UUID
				}
				if vol.Name != "" {
					reqVol.Name = &vol.Name
				}
				if vol.SizeMB > 0 {
					reqVol.SizeMb = &vol.SizeMB
				}
				if vol.Readonly {
					reqVol.Readonly = &vol.Readonly
				}
				req.Volumes = append(req.Volumes, reqVol)
			}
		case "service.uuid":
			uuid := field.Create.Set.(string)
			if uuid != "" {
				if req.ServiceGroup == nil {
					req.ServiceGroup = &platform.CreateInstanceRequestServiceGroup{}
				}
				req.ServiceGroup.Uuid = &uuid
			}
		case "service.name":
			name := field.Create.Set.(string)
			if name != "" {
				if req.ServiceGroup == nil {
					req.ServiceGroup = &platform.CreateInstanceRequestServiceGroup{}
				}
				req.ServiceGroup.Name = &name
			}
		case "service.services":
			services := field.Create.Set.([]*Service)
			if len(services) > 0 {
				if req.ServiceGroup == nil {
					req.ServiceGroup = &platform.CreateInstanceRequestServiceGroup{}
				}
				for _, svc := range services {
					req.ServiceGroup.Services = append(req.ServiceGroup.Services, platform.Service{
						Port:            svc.Source,
						DestinationPort: &svc.Destination,
						Handlers:        svc.Handlers,
					})
				}
			}
		case "service.domains":
			domains := field.Create.Set.([]Domain)
			if len(domains) > 0 {
				if req.ServiceGroup == nil {
					req.ServiceGroup = &platform.CreateInstanceRequestServiceGroup{}
				}
				for _, domain := range domains {
					name := domain.Name
					if name == "" {
						name = domain.FQDN + "."
					}
					req.ServiceGroup.Domains = append(req.ServiceGroup.Domains, platform.CreateInstanceRequestDomain{
						Name: name,
					})
				}
			}
		case "service.soft-limit":
			limit := field.Create.Set.(uint32)
			if limit > 0 {
				if req.ServiceGroup == nil {
					req.ServiceGroup = &platform.CreateInstanceRequestServiceGroup{}
				}
				req.ServiceGroup.SoftLimit = &limit
			}
		case "service.hard-limit":
			limit := field.Create.Set.(uint32)
			if limit > 0 {
				if req.ServiceGroup == nil {
					req.ServiceGroup = &platform.CreateInstanceRequestServiceGroup{}
				}
				req.ServiceGroup.HardLimit = &limit
			}
		case "autostart":
			autostart := field.Create.Set.(bool)
			req.Autostart = &autostart
		case "replicas":
			replicas := field.Create.Set.(int64)
			req.Replicas = &replicas
		case "wait_timeout_ms":
			timeout := field.Create.Set.(int64)
			req.WaitTimeoutMs = &timeout
		case "features":
			features := field.Create.Set.([]string)
			for _, f := range features {
				req.Features = append(req.Features, platform.CreateInstanceRequestFeatures(f))
			}
		}
	}

	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	keys, err := multimetro.DoMetro(ctx, cl, metro, func(ctx context.Context, mc *multimetro.MetroClient) (multimetro.Keys, error) {
		log.G(ctx).Trace().Msg("creating instance")
		resp, err := mc.CreateInstance(ctx, req)
		if err != nil {
			return nil, err
		}
		if len(resp.Data.Instances) == 0 {
			return nil, fmt.Errorf("no instances created")
		}
		created := make(multimetro.Keys, 0, len(resp.Data.Instances))
		for _, instance := range resp.Data.Instances {
			key := multimetro.Key{
				Metro: mc.Metro.Name,
				UUID:  ptr.ZeroIfNil(instance.Uuid),
				Name:  ptr.ZeroIfNil(instance.Name),
			}
			created = append(created, key)
		}
		return created, nil
	})
	if err != nil {
		return nil, err
	}
	results, err := Instance{}.Get(ctx, keys.Strings())
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (Instance) Examples() map[cmd.CmdType][]kingkong.Example {
	return map[cmd.CmdType][]kingkong.Example{
		cmd.CmdTypeGet: {
			{
				Description: "Inspect an instance by name or UUID",
				Commands:    []string{"unikraft instance get demo-instance"},
			},
		},
		cmd.CmdTypeList: {
			{
				Description: "List instances across metros",
				Commands:    []string{"unikraft instance list"},
			},
		},
		cmd.CmdTypeCreate: {
			{
				Description: "Create a new instance",
				Commands: []string{
					`unikraft instance create \
  --set name=demo-instance \
  --set metro=fra \
  --set image=nginx:latest \
  --set autostart=true \
  --set resources.memory=128 \
  --set resources.vcpus=1`,
				},
			},
		},
		cmd.CmdTypeEdit: {
			{
				Description: "Resize instance memory",
				Commands:    []string{"unikraft instance edit demo-instance --set resources.memory=256"},
			},
		},
		cmd.CmdTypeDelete: {
			{
				Description: "Delete an instance by name or UUID",
				Commands:    []string{"unikraft instance delete demo-instance"},
			},
		},
	}
}

type InstancesLogsCmd struct {
	Name []string `arg:"" completion-predictor:"resource-key-instance" help:"Names of the instances to fetch logs for."`

	Tail   int  `help:"Number of lines to show from the end of the logs."`
	Follow bool `short:"f" help:"Follow log output."`
}

func (cmd InstancesLogsCmd) Examples() []kingkong.Example {
	return []kingkong.Example{
		{
			Description: "Fetch logs from an instance",
			Commands: []string{
				"unikraft instance logs my-instance",
			},
		},
		{
			Description: "Fetch the last 100 lines of logs from an instance",
			Commands: []string{
				"unikraft instance logs my-instance --tail 100",
			},
		},
		{
			Description: "Follow logs from an instance in real-time",
			Commands: []string{
				"unikraft instance logs my-instance --follow",
			},
		},
	}
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
	Name []string `arg:"" completion-predictor:"resource-key-instance" help:"Names of the instances to start."`

	cmd.FormatOpts
}

func (cmd InstancesStartCmd) Examples() []kingkong.Example {
	return []kingkong.Example{
		{
			Description: "Start an instance",
			Commands: []string{
				"unikraft instance start demo-instance",
			},
		},
	}
}

func (c *InstancesStartCmd) Run(ctx context.Context) error {
	cfg := config.FromContextOrDefault(ctx)

	keys := multimetro.ParseKeys(c.Name)
	before, err := Instance{}.Get(ctx, keys.Strings())
	if err != nil {
		return err
	}
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return err
	}
	var targetKeys []multimetro.Key
	for _, res := range before {
		targetKeys = append(targetKeys, res.(Instance).key())
	}

	started, err := startInstances(ctx, cl, targetKeys)
	if err != nil {
		return err
	}
	updated, err := Instance{}.Get(ctx, started.Strings())
	if err != nil {
		return err
	}
	return cmd.Diff(cfg.Stdout, c.FormatOpts, Instance{}, before, updated)
}

type InstancesStopCmd struct {
	Name []string `arg:"" completion-predictor:"resource-key-instance" help:"Names of the instances to stop."`
	StopOpts

	cmd.FormatOpts
}

func (cmd InstancesStopCmd) Examples() []kingkong.Example {
	return []kingkong.Example{
		{
			Description: "Stop an instance",
			Commands: []string{
				"unikraft instance stop demo-instance",
			},
		},
		{
			Description: "Stop with a drain timeout",
			Commands: []string{
				"unikraft instance stop demo-instance --drain-timeout 30000",
			},
		},
		{
			Description: "Force stop an instance",
			Commands: []string{
				"unikraft instance stop demo-instance --force",
			},
		},
	}
}

func (c *InstancesStopCmd) Run(ctx context.Context) error {
	cfg := config.FromContextOrDefault(ctx)

	keys := multimetro.ParseKeys(c.Name)
	before, err := Instance{}.Get(ctx, keys.Strings())
	if err != nil {
		return err
	}
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return err
	}
	var targetKeys []multimetro.Key
	for _, res := range before {
		targetKeys = append(targetKeys, res.(Instance).key())
	}
	if len(targetKeys) == 0 {
		targetKeys = keys
	}
	stopped, err := stopInstances(ctx, cl, targetKeys, c.StopOpts)
	if err != nil {
		return err
	}
	updated, err := Instance{}.Get(ctx, stopped.Strings())
	if err != nil {
		return err
	}
	return cmd.Diff(cfg.Stdout, c.FormatOpts, Instance{}, before, updated)
}

type InstancesRestartCmd struct {
	Name []string `arg:"" completion-predictor:"resource-key-instance" help:"Names of the instances to restart."`
	StopOpts

	cmd.FormatOpts
}

func (cmd InstancesRestartCmd) Examples() []kingkong.Example {
	return []kingkong.Example{
		{
			Description: "Restart an instance",
			Commands: []string{
				"unikraft instance restart demo-instance",
			},
		},
		{
			Description: "Force restart an instance",
			Commands: []string{
				"unikraft instance restart demo-instance --force",
			},
		},
	}
}

func (c *InstancesRestartCmd) Run(ctx context.Context) error {
	cfg := config.FromContextOrDefault(ctx)

	keys := multimetro.ParseKeys(c.Name)
	before, err := Instance{}.Get(ctx, keys.Strings())
	if err != nil {
		return err
	}
	cl, err := multimetro.NewClient(ctx)
	if err != nil {
		return err
	}
	var targetKeys []multimetro.Key
	for _, res := range before {
		targetKeys = append(targetKeys, res.(Instance).key())
	}

	stopped, err := stopInstances(ctx, cl, targetKeys, c.StopOpts)
	if err != nil {
		return err
	}
	started, err := startInstances(ctx, cl, stopped)
	if err != nil {
		return err
	}
	updated, err := Instance{}.Get(ctx, started.Strings())
	if err != nil {
		return err
	}
	return cmd.Diff(cfg.Stdout, c.FormatOpts, Instance{}, before, updated)
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

func startInstances(ctx context.Context, cl *multimetro.Client, keys multimetro.Keys) (multimetro.Keys, error) {
	started, err := multimetro.DoKeys(ctx, cl, keys, func(ctx context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]multimetro.Key, []multimetro.Key, error) {
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
		return started, started, nil
	})
	if err != nil {
		return nil, err
	}
	return multimetro.Keys(started), nil
}

func stopInstances(ctx context.Context, cl *multimetro.Client, keys multimetro.Keys, opts StopOpts) (multimetro.Keys, error) {
	stopped, err := multimetro.DoKeys(ctx, cl, keys, func(ctx context.Context, mc *multimetro.MetroClient, keys multimetro.Keys) ([]multimetro.Key, []multimetro.Key, error) {
		log.G(ctx).Trace().Msg("stopping instances")
		reqs := make([]platform.StopInstancesRequestItem, 0, len(keys))
		for _, key := range keys {
			reqs = append(reqs, opts.toReq(key.NameOrUUID()))
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
		return nil, err
	}
	return multimetro.Keys(stopped), nil
}
