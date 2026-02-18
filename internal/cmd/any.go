// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"unikraft.com/cli/internal/resource"
	resourcecmd "unikraft.com/cli/internal/resource/cmd"
)

type AnyResourceCmd struct {
	resourcecmd.ResourceCmd[AnyResource]
	resourcecmd.GettableResourceCmd[AnyResource]  `set:"name=resource" set:"names=resources"`
	resourcecmd.ListableResourceCmd[AnyResource]  `set:"name=resource" set:"names=resources"`
	resourcecmd.EditableResourceCmd[AnyResource]  `set:"name=resource" set:"names=resources"`
	resourcecmd.CreatableResourceCmd[AnyResource] `set:"name=resource" set:"names=resources"`
	resourcecmd.DeletableResourceCmd[AnyResource] `set:"name=resource" set:"names=resources"`
	resourcecmd.PurgeableResourceCmd[AnyResource] `set:"name=resource" set:"names=resources"`
}

var resourceBackends = []resource.Resource{
	Instance{},
	Volume{},
	ServiceGroup{},
	Certificate{},
}

func backendByType(typ string) (resource.Resource, bool) {
	for _, backend := range resourceBackends {
		if backend.Type().Name == typ {
			return backend, true
		}
	}
	return nil, false
}

// AnyResource is a special resource that multiplexes to different backend
// resources based on the key prefix (e.g., "instance:", "volume:", etc.).
// When populated, resources have their full fields. When empty (header field),
// it only has "type" and "key" fields.
type AnyResource struct {
	Type_ string `field:"type,short" create:"set,required"`
	Key_  string `field:"key,short"`

	// The actual underlying resource, if populated
	underlying resource.Resource
}

func (a AnyResource) Type() resource.Type {
	if a.underlying != nil {
		return a.underlying.Type()
	}
	return resource.Type{
		Name:  "resource",
		Names: "resources",
	}
}

type anyResourceKey struct {
	typ string
	key string
}

func (k anyResourceKey) String() string {
	if k.typ == "" {
		return k.key
	}
	return k.typ + ":" + k.key
}

func (k *anyResourceKey) UnmarshalText(text []byte) error {
	typ, key, found := strings.Cut(string(text), ":")
	if found {
		k.typ = typ
		k.key = key
	} else {
		k.key = string(text)
	}
	return nil
}

func (a AnyResource) Key() resource.Key {
	if a.underlying != nil {
		return a.underlying.Key()
	}
	return anyResourceKey{
		typ: a.Type_,
		key: a.Key_,
	}
}

func (a AnyResource) Fields() ([]resource.Field, error) {
	fields, err := resource.FieldsFromStruct(a)
	if err != nil {
		return nil, err
	}
	underlying := a.underlying
	if underlying == nil && a.Type_ != "" {
		if backend, ok := backendByType(a.Type_); ok {
			underlying = backend
		}
	}
	if underlying != nil {
		underlyingFields, err := underlying.Fields()
		if err != nil {
			return nil, err
		}
		fields = append(fields, underlyingFields...)
	}
	return fields, nil
}

func (a AnyResource) WithType(typ string) resource.Resource {
	a.Type_ = typ
	return a
}

func (a AnyResource) Raw() any {
	if a.underlying != nil {
		return a.underlying.Raw()
	}
	return nil
}

func (a AnyResource) Get(ctx context.Context, keys []string) ([]resource.Resource, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys provided")
	}

	requests := make(map[string][]string)
	for _, key := range keys {
		var k anyResourceKey
		if err := k.UnmarshalText([]byte(key)); err != nil {
			return nil, fmt.Errorf("invalid resource key %q: %w", key, err)
		}
		if k.typ == "" {
			return nil, fmt.Errorf("resource key %q must include resource type prefix", key)
		}

		backend, ok := backendByType(k.typ)
		if !ok {
			return nil, fmt.Errorf("unknown resource type: %s", k.typ)
		}
		if _, ok := backend.(resource.GettableResource); !ok {
			return nil, fmt.Errorf("resource type %s does not support Get", k.typ)
		}
		requests[k.typ] = append(requests[k.typ], k.key)
	}

	var (
		results []resource.Resource
		mu      sync.Mutex
	)

	eg, ctx := errgroup.WithContext(ctx)

	for typ, keyList := range requests {
		if len(keyList) == 0 {
			continue
		}
		backend, ok := backendByType(typ)
		if !ok {
			return nil, fmt.Errorf("unknown resource type: %s", typ)
		}
		gettable := backend.(resource.GettableResource)
		typ := typ
		keyList := keyList
		eg.Go(func() error {
			resources, err := gettable.Get(ctx, keyList)
			if err != nil {
				return fmt.Errorf("failed to get %s resources: %w", typ, err)
			}

			mu.Lock()
			defer mu.Unlock()
			for _, res := range resources {
				key := res.Key().String()
				results = append(results, AnyResource{
					Type_:      res.Type().Name,
					Key_:       res.Type().Name + ":" + key,
					underlying: res,
				})
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

func (a AnyResource) List(ctx context.Context) ([]resource.Resource, error) {
	var results []resource.Resource

	for _, backend := range resourceBackends {
		listable, ok := backend.(resource.ListableResource)
		if !ok {
			continue
		}

		resources, err := listable.List(ctx)
		if err != nil {
			continue
		}

		for _, res := range resources {
			results = append(results, AnyResource{
				Type_:      res.Type().Name,
				Key_:       res.Type().Name + ":" + res.Key().String(),
				underlying: res,
			})
		}
	}

	return results, nil
}

func (a AnyResource) Edit(ctx context.Context, target resource.Resource, fields []resource.Field) (resource.Resource, error) {
	anyRes, ok := target.(AnyResource)
	if !ok {
		return nil, fmt.Errorf("expected AnyResource, got %T", target)
	}
	if anyRes.underlying == nil {
		return nil, fmt.Errorf("cannot edit resource without underlying resource")
	}

	typ := anyRes.underlying.Type().Name
	backend, ok := backendByType(typ)
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", typ)
	}

	editable, ok := backend.(resource.EditableResource)
	if !ok {
		return nil, fmt.Errorf("resource type %s does not support Edit", typ)
	}

	result, err := editable.Edit(ctx, anyRes.underlying, fields)
	if err != nil {
		return nil, fmt.Errorf("failed to edit %s resource: %w", typ, err)
	}

	return AnyResource{
		Type_:      result.Type().Name,
		Key_:       result.Type().Name + ":" + result.Key().String(),
		underlying: result,
	}, nil
}

func (a AnyResource) Create(ctx context.Context, fields []resource.Field) ([]resource.Resource, error) {
	var typ string
	for _, field := range fields {
		if field.Name == "type" {
			if s, ok := field.Value.(string); ok {
				typ = s
			}
			break
		}
	}

	if typ == "" {
		return nil, fmt.Errorf("resource type must be specified in fields")
	}

	backend, ok := backendByType(typ)
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", typ)
	}

	creatable, ok := backend.(resource.CreatableResource)
	if !ok {
		return nil, fmt.Errorf("resource type %s does not support Create", typ)
	}

	results, err := creatable.Create(ctx, a.underlyingFields(fields))
	if err != nil {
		return nil, fmt.Errorf("failed to create %s resource: %w", typ, err)
	}

	var wrapped []resource.Resource
	for _, res := range results {
		wrapped = append(wrapped, AnyResource{
			Type_:      res.Type().Name,
			Key_:       res.Type().Name + ":" + res.Key().String(),
			underlying: res,
		})
	}

	return wrapped, nil
}

func (a AnyResource) underlyingFields(fields []resource.Field) []resource.Field {
	filtered := make([]resource.Field, 0, len(fields))
	for _, field := range fields {
		if field.Name == "type" || field.Name == "key" {
			continue
		}
		filtered = append(filtered, field)
	}
	return filtered
}

func (a AnyResource) Delete(ctx context.Context, targets []resource.Resource) error {
	targetsByType := make(map[string][]resource.Resource)
	for _, target := range targets {
		anyRes, ok := target.(AnyResource)
		if !ok {
			return fmt.Errorf("expected AnyResource, got %T", target)
		}
		if anyRes.underlying == nil {
			return fmt.Errorf("cannot delete resource without underlying resource")
		}

		typ := anyRes.underlying.Type().Name
		targetsByType[typ] = append(targetsByType[typ], anyRes.underlying)
	}

	for typ, typeTargets := range targetsByType {
		backend, ok := backendByType(typ)
		if !ok {
			return fmt.Errorf("unknown resource type: %s", typ)
		}

		deletable, ok := backend.(resource.DeletableResource)
		if !ok {
			return fmt.Errorf("resource type %s does not support Delete", typ)
		}

		if err := deletable.Delete(ctx, typeTargets); err != nil {
			return fmt.Errorf("failed to delete %s resources: %w", typ, err)
		}
	}

	return nil
}
