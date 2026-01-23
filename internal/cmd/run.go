// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"
	"fmt"
	"io"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/logs"
	"unikraft.com/cli/internal/multimetro"
	"unikraft.com/cli/internal/resource"
	"unikraft.com/cli/internal/resource/cmd"
	"unikraft.com/cli/internal/resource/patch"
)

type RunCmd struct {
	Image string   `arg:"" help:"Image to run."`
	Args  []string `arg:"" optional:"" help:"Arguments to pass to the instance."`

	Name  string `short:"n" help:"Name of the instance."`
	Metro string `help:"Metro to deploy the instance in." required:"" placeholder:"metro"`

	Env    []string `short:"e" help:"Set environment variables (KEY=VALUE)."`
	Memory int      `short:"m" help:"Memory in MB."`
	Volume []string `short:"v" help:"Attach a volume (NAME:AT[:ro] or NAME:SIZE:AT[:ro] or UUID:AT[:ro])."`
	Vcpus  int      `help:"Number of vCPUs."`

	DryRun bool `help:"Show the create preview without executing."`

	Service     string   `help:"Service group name."`
	Publish     []string `short:"p" help:"Publish a service port (SOURCE:DESTINATION[/HANDLERS])."`
	Domain      []string `help:"Add a service domain (FQDN)."`
	ScaleToZero []string `help:"Enable scale-to-zero."`

	Restart       string   `help:"Restart policy."`
	Autostart     bool     `help:"Start the instance automatically." default:"true"`
	Replicas      int64    `help:"Number of replicas."`
	WaitTimeoutMs int64    `help:"Wait timeout in milliseconds."`
	Features      []string `help:"Enable instance features."`

	Follow bool `short:"f" help:"Follow instance logs after creation."`
}

func (c *RunCmd) Run(ctx context.Context, cfg *config.Config, sandbox *resource.Sandbox) error {
	if c.Image == "" {
		return fmt.Errorf("image is required")
	}
	if c.Metro == "" {
		return fmt.Errorf("metro is required")
	}

	env, err := patch.ParseNewValue[map[string]string](c.Env)
	if err != nil {
		return err
	}
	volumes, err := patch.ParseNewValue[[]*InstanceVolume](c.Volume)
	if err != nil {
		return err
	}
	services, err := patch.ParseNewValue[[]*Service](c.Publish)
	if err != nil {
		return err
	}
	domains, err := patch.ParseNewValue[[]Domain](c.Domain)
	if err != nil {
		return err
	}
	scaleToZero, err := patch.ParseNewValue[*InstanceScaleToZero](c.ScaleToZero)
	if err != nil {
		return err
	}

	fields, err := Instance{}.Fields()
	if err != nil {
		return err
	}
	if err := c.applyCreatePatches(fields, env, volumes, services, domains, scaleToZero); err != nil {
		return err
	}

	if c.DryRun {
		return cmd.PrintPatches(cfg.Stdout, fields, true)
	}

	var creatable resource.CreatableResource = Instance{}
	if sandbox != nil {
		creatable = sandbox.WrapCreatable(creatable)
	}
	created, err := creatable.Create(ctx, fields)
	if err != nil {
		return err
	}
	instance := created.(Instance)
	fmt.Fprintln(cfg.Stdout, created.Key())

	if c.Follow {
		cl, err := multimetro.NewClient(ctx)
		if err != nil {
			return err
		}
		_, err = multimetro.DoKeyExact(ctx, cl, instance.key(), func(ctx context.Context, mc *multimetro.MetroClient) (any, error) {
			r, err := logs.InstanceLogs(ctx, mc).Reader(instance.key().NameOrUUID(), 0, true)
			if err != nil {
				return nil, err
			}
			_, err = io.Copy(cfg.Stdout, r)
			return nil, err
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *RunCmd) applyCreatePatches(fields []resource.Field, env map[string]string, volumes []*InstanceVolume, services []*Service, domains []Domain, scaleToZero *InstanceScaleToZero) error {
	patches := c.runCreatePatches(env, volumes, services, domains, scaleToZero)
	for path, field := range resource.IterFields(fields) {
		if field.Create == nil {
			continue
		}
		field.Create = nil
		if value, ok := patches[path.String()]; ok {
			field.Create = &resource.Patch{Set: value}
		}
	}

	return nil
}

func (c *RunCmd) runCreatePatches(env map[string]string, volumes []*InstanceVolume, services []*Service, domains []Domain, scaleToZero *InstanceScaleToZero) map[string]any {
	patches := map[string]any{
		// FIXME: parse image key, don't require exact matches
		"image": c.Image,
		"metro": c.Metro,
	}
	if c.Name != "" {
		patches["name"] = c.Name
	}
	if len(c.Args) > 0 {
		patches["runtime.args"] = c.Args
	}
	if len(env) > 0 {
		patches["runtime.env"] = env
	}
	if c.Memory > 0 {
		patches["resources.memory"] = c.Memory
	}
	if c.Vcpus > 0 {
		patches["resources.vcpus"] = c.Vcpus
	}
	if c.Restart != "" {
		patches["restart.policy"] = c.Restart
	}
	if scaleToZero != nil {
		if scaleToZero.Enabled != nil {
			patches["scale-to-zero.enabled"] = *scaleToZero.Enabled
		} else {
			patches["scale-to-zero.enabled"] = true
		}
		if scaleToZero.Policy != "" {
			patches["scale-to-zero.policy"] = scaleToZero.Policy
		}
		if scaleToZero.Stateful {
			patches["scale-to-zero.stateful"] = scaleToZero.Stateful
		}
		if scaleToZero.CooldownTime > 0 {
			patches["scale-to-zero.cooldown-time"] = scaleToZero.CooldownTime
		}
	}
	if len(volumes) > 0 {
		patches["volumes"] = volumes
	}
	if c.Service != "" {
		key := multimetro.ParseKey(c.Service)
		if key.UUID != "" {
			patches["service.uuid"] = key.UUID
		} else {
			patches["service.name"] = key.Name
		}
	}
	if len(services) > 0 {
		patches["service.services"] = services
	}
	if len(domains) > 0 {
		patches["service.domains"] = domains
	}
	if c.Autostart {
		patches["autostart"] = c.Autostart
	}
	if c.Replicas > 0 {
		patches["replicas"] = c.Replicas
	}
	if c.WaitTimeoutMs > 0 {
		patches["wait_timeout_ms"] = c.WaitTimeoutMs
	}
	if len(c.Features) > 0 {
		patches["features"] = c.Features
	}
	return patches
}
