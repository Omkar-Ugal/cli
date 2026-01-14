// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/containerd/containerd/v2/pkg/filters"
	"github.com/lunixbochs/vtclean"
	"github.com/sergi/go-diff/diffmatchpatch"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/kvwriter"
	"unikraft.com/cli/internal/prettydiff"
	"unikraft.com/cli/internal/resource"
	"unikraft.com/cli/internal/resource/patch"
)

type ResourceCmd[R resource.GettableResource] struct {
	List ResourceListCmd[R]    `cmd:"" help:"List ${names}." aliases:"ls"`
	Get  ResourceInspectCmd[R] `cmd:"" help:"Inspect a ${name}." aliases:"inspect,show"`
}
type DeletableResourceCmd[R resource.DeletableResource] struct {
	Delete ResourceRemoveCmd[R] `cmd:"" help:"Remove a ${name}." aliases:"rm,remove"`
}
type EditableResourceCmd[R resource.EditableResource] struct {
	Edit ResourceEditCmd[R] `cmd:"" help:"Edit a ${name}."`
}
type CreatableResourceCmd[R resource.CreatableResource] struct {
	Create ResourceCreateCmd[R] `cmd:"" help:"Create a ${name}."`
}

func (cmd *ResourceCmd[R]) Help() string {
	var r R
	return fmt.Sprintf("Manage %s.", r.Type().Names)
}

type FormatOpts struct {
	// FIXME: not able to pass values beginning with -
	// https://github.com/alecthomas/kong/issues/290
	Field  []string `short:"f" help:"Specify which fields to include in the output."`
	Filter []string `help:"Filter output based on a field value (e.g. --filter status==active)." sep:"none"`

	Quiet  bool   `short:"q" help:"Only display resource keys."`
	Format string `help:"Format the output using a Go template."`
	Raw    bool   `help:"Output raw JSON API response."`
}

type ResourceListCmd[R resource.GettableResource] struct {
	Name []string `arg:"" optional:"" help:"Names of the ${names} to list."`

	FormatOpts
}

func (cmd *ResourceListCmd[R]) Help() string {
	return "Lister"
}

func (cmd *ResourceListCmd[R]) Run(ctx context.Context, cfg *config.Config, sandbox *resource.Sandbox) error {
	filter, err := filters.ParseAll(cmd.Filter...)
	if err != nil {
		return err
	}
	ctx = resource.WithFilter(ctx, filter)

	var empty R
	r := sandbox.WrapGettable(empty)
	var resources []resource.Resource
	if len(cmd.Name) > 0 {
		resources, err = r.Get(ctx, cmd.Name)
	} else {
		resources, err = r.List(ctx)
	}
	if err != nil {
		return err
	}

	resources, err = filterResources(resources, filter)
	if err != nil {
		return err
	}

	switch {
	case cmd.Quiet:
		return printQuiet(cfg.Stdout, resources...)
	case cmd.Raw:
		return printRaw(cfg.Stdout, resources...)
	case cmd.Format != "":
		return printTemplate(cfg.Stdout, cmd.Format, resources...)
	default:
		return printTable[R](cfg.Stdout, cmd.Field, resources...)
	}
}

type ResourceInspectCmd[R resource.GettableResource] struct {
	Name []string `arg:"" help:"Names of the ${names} to inspect."`

	FormatOpts
}

func (cmd *ResourceInspectCmd[R]) Run(ctx context.Context, cfg *config.Config, sandbox *resource.Sandbox) error {
	filter, err := filters.ParseAll(cmd.Filter...)
	if err != nil {
		return err
	}
	ctx = resource.WithFilter(ctx, filter)

	var empty R
	r := sandbox.WrapGettable(empty)
	resources, err := r.Get(ctx, cmd.Name)
	if err != nil {
		return err
	}

	resources, err = filterResources(resources, filter)
	if err != nil {
		return err
	}

	switch {
	case cmd.Quiet:
		return printQuiet(cfg.Stdout, resources...)
	case cmd.Raw:
		return printRaw(cfg.Stdout, resources...)
	case cmd.Format != "":
		return printTemplate(cfg.Stdout, cmd.Format, resources...)
	default:
		return printInspect(cfg.Stdout, cmd.Field, resources...)
	}
}

func filterResources(resources []resource.Resource, filter filters.Filter) (filtered []resource.Resource, rerr error) {
	for _, res := range resources {
		if filter.Match(filters.AdapterFunc(func(key []string) (string, bool) {
			fields, err := res.Fields()
			if err != nil {
				if rerr == nil {
					rerr = fmt.Errorf("failed to get fields for resource %s: %w", res.Key(), err)
				}
				return "", false
			}
			fields = resource.GetFieldByPath(fields, key)
			if fields == nil {
				return "", false
			}
			if len(fields) != 1 {
				// 0 fields = no exact match
				// >1 fields = ambiguous match
				return "", false
			}
			// HACK: vtclean to remove any escape sequences from rendered output
			return vtclean.Clean(fields[0].ValueString(), false), true
		})) {
			filtered = append(filtered, res)
		}
	}
	return filtered, rerr
}

type ResourceRemoveCmd[R resource.DeletableResource] struct {
	Name []string `arg:"" help:"Names of the ${names} to remove."`
}

func (cmd *ResourceRemoveCmd[R]) Run(ctx context.Context, sandbox *resource.Sandbox) error {
	var empty R
	r := sandbox.WrapDeletable(empty)
	resources, err := r.Get(ctx, cmd.Name)
	if err != nil {
		return err
	}
	return r.Delete(ctx, resources)
}

type ResourceEditCmd[R resource.EditableResource] struct {
	Name string `arg:"" help:"Name of the ${name} to edit."`

	Set []map[string]string `help:"Key-value pairs to update the ${name} with." sep:"none" mapsep:"none"`
	Add []map[string]string `help:"Key-value pairs to add to the ${name}." sep:"none" mapsep:"none"`
	Del []map[string]string `help:"Keys to delete from the ${name}." sep:"none" mapsep:"none"`

	Visual bool `short:"e" help:"Open an editor to modify fields visually."`
}

func (cmd *ResourceEditCmd[R]) toPatchSpec() patch.PatchSpec {
	spec := patch.PatchSpec{
		Set: make(map[string][]string),
		Add: make(map[string][]string),
		Del: make(map[string][]string),
	}
	for _, m := range cmd.Set {
		for k, v := range m {
			spec.Set[k] = append(spec.Set[k], v)
		}
	}
	for _, m := range cmd.Add {
		for k, v := range m {
			spec.Add[k] = append(spec.Add[k], v)
		}
	}
	for _, m := range cmd.Del {
		for k, v := range m {
			spec.Del[k] = append(spec.Del[k], v)
		}
	}
	return spec
}

func (cmd *ResourceEditCmd[R]) Run(ctx context.Context, cfg *config.Config, sandbox *resource.Sandbox) error {
	var empty R
	r := sandbox.WrapEditable(empty)
	resources, err := r.Get(ctx, []string{cmd.Name})
	if err != nil {
		return err
	}
	if len(resources) == 0 {
		return fmt.Errorf("resource not found: %s", cmd.Name)
	}
	if len(resources) > 1 {
		var keys []string
		for _, res := range resources {
			keys = append(keys, res.Key())
		}
		return fmt.Errorf("ambiguous resource name: %s (found %v)", cmd.Name, keys)
	}
	res := resources[0]

	spec := cmd.toPatchSpec()

	// Check for duplicate field paths across Set/Add/Del
	allFields := make(map[string]int)
	for k := range spec.Set {
		allFields[k]++
	}
	for k := range spec.Add {
		allFields[k]++
	}
	for k := range spec.Del {
		allFields[k]++
	}
	for k, count := range allFields {
		if count > 1 {
			return fmt.Errorf("field %s has multiple patch operations", k)
		}
	}

	fields, err := res.Fields()
	if err != nil {
		return fmt.Errorf("failed to get fields: %w", err)
	}
	patched, err := patch.PatchedFields(fields, spec)
	if err != nil {
		return err
	}
	if cmd.Visual {
		patched, err = patch.VisualEdit(ctx, cfg, fields, patched)
		if err != nil {
			return err
		}
	}
	patched = patch.FilterPatchableFields(patched)

	start := &bytes.Buffer{}
	err = printKV(start, nil, res)
	if err != nil {
		return err
	}
	if len(patched) > 0 {
		res, err = r.Edit(ctx, res, patched)
		if err != nil {
			return err
		}
	}
	end := &bytes.Buffer{}
	err = printKV(end, nil, res)
	if err != nil {
		return err
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(start.String(), end.String(), false)
	tw := kvwriter.KeyValueWriter(cfg.Stdout, "  ")
	_, err = io.Copy(tw, strings.NewReader(prettydiff.Render(diffs)))
	if err != nil {
		return err
	}
	return tw.Flush()
}

type ResourceCreateCmd[R resource.CreatableResource] struct {
	Set []map[string]string `help:"Key-value pairs for creating the ${name}." sep:"none" mapsep:"none"`

	Visual bool `short:"e" help:"Open an editor to set fields visually."`
}

func (cmd *ResourceCreateCmd[R]) toPatchSpec() patch.PatchSpec {
	spec := patch.PatchSpec{
		Create: true,
		Set:    make(map[string][]string),
	}
	for _, m := range cmd.Set {
		for k, v := range m {
			spec.Set[k] = append(spec.Set[k], v)
		}
	}
	return spec
}

func (cmd *ResourceCreateCmd[R]) Run(ctx context.Context, cfg *config.Config, sandbox *resource.Sandbox) error {
	var empty R
	r := sandbox.WrapCreatable(empty)
	fields, err := r.Fields()
	if err != nil {
		return fmt.Errorf("failed to get fields: %w", err)
	}
	patched, err := patch.PatchedFields(fields, cmd.toPatchSpec())
	if err != nil {
		return err
	}

	if cmd.Visual {
		// FIXME: should allow required fields
		patched, err = patch.VisualCreate(ctx, cfg, fields, patched)
		if err != nil {
			return err
		}
	}

	fields = patch.FilterCreatableFields(patched)

	res, err := r.Create(ctx, fields)
	if err != nil {
		return err
	}
	return printInspect(cfg.Stdout, nil, res)
}
