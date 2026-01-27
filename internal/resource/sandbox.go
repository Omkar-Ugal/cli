// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"unikraft.com/x/log"

	xmaps "unikraft.com/cli/internal/x/maps"
)

// Sandbox represents a testing sandbox for resources. Resources created in the
// sandbox are tracked and isolated from other resources (to provide our
// testing framework a reliable clean environment).
//
// Sandboxes are persisted to disk as JSON files, which track resource types
// and keys that belong to the sandbox.
type Sandbox struct {
	Path    string
	Keys    map[string]map[string]struct{}
	Cleanup []Resource
}

const UnikraftSandboxEnv = "UNIKRAFT_X_SANDBOX"

func LoadSandboxFromEnv(resources ...Resource) (*Sandbox, error) {
	path, ok := os.LookupEnv(UnikraftSandboxEnv)
	if !ok {
		return nil, nil
	}
	sandbox, err := LoadSandbox(path, resources...)
	if err != nil {
		return nil, fmt.Errorf("failed to load sandbox from %s: %w", path, err)
	}
	return sandbox, nil
}

func LoadSandbox(path string, resources ...Resource) (*Sandbox, error) {
	s := Sandbox{Path: path, Cleanup: resources}
	s.Keys = make(map[string]map[string]struct{})
	for _, r := range resources {
		if _, ok := s.Keys[r.Type().Name]; !ok {
			s.Keys[r.Type().Name] = make(map[string]struct{})
		}
	}

	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return &s, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to open sandbox file: %w", err)
	}
	defer f.Close()

	var keys map[string][]string
	err = json.NewDecoder(f).Decode(&keys)
	if err != nil {
		return nil, fmt.Errorf("failed to decode sandbox file: %w", err)
	}
	for rtype, rkeys := range keys {
		if _, ok := s.Keys[rtype]; !ok {
			continue
		}
		for _, rkey := range rkeys {
			s.Keys[rtype][rkey] = struct{}{}
		}
	}

	return &s, nil
}

func (s *Sandbox) Save() error {
	if s == nil {
		return nil
	}
	f, err := os.Create(s.Path)
	if err != nil {
		return fmt.Errorf("failed to create sandbox file: %w", err)
	}
	defer f.Close()

	keys := make(map[string][]string, len(s.Keys))
	for rtype, rkeys := range s.Keys {
		keys[rtype] = xmaps.OrderedKeys(rkeys)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(keys)
	if err != nil {
		return fmt.Errorf("failed to encode sandbox file: %w", err)
	}

	return nil
}

// Teardown attempts to delete all resources tracked by the sandbox. Some
// resources may not be deletable, in which case they are skipped.
func (s *Sandbox) Teardown(ctx context.Context) (rerr error) {
	if s == nil {
		return nil
	}
	log.G(ctx).Debug().
		Str("path", s.Path).
		Msg("tearing down sandbox")
	for _, r := range s.Cleanup {
		name := r.Type().Name
		r, ok := r.(DeletableResource)
		if !ok {
			log.G(ctx).Debug().
				Str("resource", name).
				Msg("skipping resource cleanup as it is not deletable")
			continue
		}

		targets := xmaps.OrderedKeys(s.Keys[name])
		if len(targets) == 0 {
			log.G(ctx).Debug().
				Str("resource", name).
				Msg("no resources to clean up in sandbox")
			continue
		}

		log.G(ctx).Debug().
			Str("resource", name).
			Strs("targets", targets).
			Msg("cleaning up resources in sandbox")

		resources, err := r.Get(ctx, targets)
		if err != nil {
			if strings.Contains(err.Error(), "keys not found") {
				continue // HACK: some resources may have been deleted through other mysterious means
			}
			rerr = errors.Join(rerr, fmt.Errorf("failed to get resources for cleanup: %w", err))
			continue
		}
		err = r.Delete(ctx, resources)
		if err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to delete resources for cleanup: %w", err))
			continue
		}
	}
	return rerr
}

func (s *Sandbox) Add(ctx context.Context, r Resource) error {
	if s == nil {
		return nil
	}
	if _, ok := s.Keys[r.Type().Name]; !ok {
		return nil
	}
	s.Keys[r.Type().Name][r.Key().String()] = struct{}{}

	fields, err := r.Fields()
	if err != nil {
		return fmt.Errorf("failed to get fields for resource %s: %w", r.Key(), err)
	}
	for _, field := range IterFields(fields) {
		for _, link := range field.Links {
			for _, r := range s.Cleanup {
				if r.Type().Name != link.Type {
					continue
				}

				r, ok := r.(GettableResource)
				if !ok {
					continue
				}
				linkedResources, err := r.Get(ctx, []string{link.Key})
				if err != nil {
					return fmt.Errorf("failed to get linked resource %s %s: %w", link.Type, link.Key, err)
				}
				for _, linkedResource := range linkedResources {
					if err := s.Add(ctx, linkedResource); err != nil {
						return err
					}
				}
				break
			}
		}
	}

	return nil
}

func (s *Sandbox) Remove(r Resource) {
	if s == nil {
		return
	}
	if _, ok := s.Keys[r.Type().Name]; !ok {
		return
	}
	delete(s.Keys[r.Type().Name], r.Key().String())
}

func (s *Sandbox) Has(r Resource) bool {
	if s == nil {
		return true
	}
	if _, ok := s.Keys[r.Type().Name]; !ok {
		return true
	}
	_, ok := s.Keys[r.Type().Name][r.Key().String()]
	return ok
}

func (s *Sandbox) Missing(r Resource) bool {
	return !s.Has(r)
}

func (s *Sandbox) WrapGettable(r GettableResource) GettableResource {
	if s == nil {
		return r
	}
	return sandboxedGettableResource{
		GettableResource: r,
		sandbox:          s,
	}
}

func (s *Sandbox) WrapListable(r ListableResource) ListableResource {
	if s == nil {
		return r
	}
	return sanboxedListableResource{
		ListableResource: r,
		sandbox:          s,
	}
}

func (s *Sandbox) WrapEditable(r EditableResource) EditableResource {
	if s == nil {
		return r
	}
	return sandboxedEditableResource{
		EditableResource: r,
		sandbox:          s,
	}
}

func (s *Sandbox) WrapCreatable(r CreatableResource) CreatableResource {
	if s == nil {
		return r
	}
	return sandboxedCreatableResource{
		CreatableResource: r,
		sandbox:           s,
	}
}

func (s *Sandbox) WrapDeletable(r DeletableResource) DeletableResource {
	if s == nil {
		return r
	}
	return sandboxedDeletableResource{
		DeletableResource: r,
		sandbox:           s,
	}
}

type sandboxedGettableResource struct {
	GettableResource
	sandbox *Sandbox
}

func (r sandboxedGettableResource) Get(ctx context.Context, keys []string) ([]Resource, error) {
	resources, err := r.GettableResource.Get(ctx, keys)
	if err != nil {
		return nil, err
	}
	resources = slices.DeleteFunc(resources, r.sandbox.Missing)
	if len(resources) == 0 {
		return nil, fmt.Errorf("no resources found in the sandbox")
	}
	return resources, nil
}

type sanboxedListableResource struct {
	ListableResource
	sandbox *Sandbox
}

func (r sanboxedListableResource) List(ctx context.Context) ([]Resource, error) {
	resources, err := r.ListableResource.List(ctx)
	if err != nil {
		return nil, err
	}
	resources = slices.DeleteFunc(resources, r.sandbox.Missing)
	return resources, nil
}

type sandboxedEditableResource struct {
	EditableResource
	sandbox *Sandbox
}

func (r sandboxedEditableResource) Get(ctx context.Context, keys []string) ([]Resource, error) {
	return sandboxedGettableResource{
		GettableResource: r.EditableResource,
		sandbox:          r.sandbox,
	}.Get(ctx, keys)
}

func (r sandboxedEditableResource) Edit(ctx context.Context, target Resource, fields []Field) (Resource, error) {
	if r.sandbox.Missing(target) {
		return nil, fmt.Errorf("resource %s is not in the sandbox", target.Key())
	}
	return r.EditableResource.Edit(ctx, target, fields)
}

type sandboxedCreatableResource struct {
	CreatableResource
	sandbox *Sandbox
}

func (r sandboxedCreatableResource) Get(ctx context.Context, keys []string) ([]Resource, error) {
	return sandboxedGettableResource{
		GettableResource: r.CreatableResource,
		sandbox:          r.sandbox,
	}.Get(ctx, keys)
}

func (r sandboxedCreatableResource) Create(ctx context.Context, fields []Field) ([]Resource, error) {
	resources, err := r.CreatableResource.Create(ctx, fields)
	if err != nil {
		return nil, err
	}
	for _, res := range resources {
		if err := r.sandbox.Add(ctx, res); err != nil {
			return nil, err
		}
	}
	return resources, nil
}

type sandboxedDeletableResource struct {
	DeletableResource
	sandbox *Sandbox
}

func (r sandboxedDeletableResource) Get(ctx context.Context, keys []string) ([]Resource, error) {
	return sandboxedGettableResource{
		GettableResource: r.DeletableResource,
		sandbox:          r.sandbox,
	}.Get(ctx, keys)
}

func (r sandboxedDeletableResource) Delete(ctx context.Context, targets []Resource) error {
	for _, res := range targets {
		if r.sandbox.Missing(res) {
			return fmt.Errorf("resource %s is not in the sandbox", res.Key())
		}
	}
	err := r.DeletableResource.Delete(ctx, targets)
	if err != nil {
		return err
	}
	for _, res := range targets {
		r.sandbox.Remove(res)
	}
	return nil
}
