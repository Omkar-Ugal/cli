// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").

package resource

type Registry struct {
	entries []*ResourceDescriptor
	byName  map[string]*ResourceDescriptor
}

type ResourceDescriptor struct {
	Name  string
	Names string
	List  ListableResource
	Get   GettableResource
}

func NewRegistry() *Registry {
	return &Registry{
		byName: make(map[string]*ResourceDescriptor),
	}
}

func (r *Registry) Register(list ListableResource, get GettableResource) *ResourceDescriptor {
	if list == nil {
		return nil
	}
	info := list.Type()
	desc := &ResourceDescriptor{
		Name:  info.Name,
		Names: info.Names,
		List:  list,
		Get:   get,
	}
	r.entries = append(r.entries, desc)
	r.byName[info.Name] = desc
	r.byName[info.Names] = desc
	return desc
}

func (r *Registry) Entries() []*ResourceDescriptor {
	return r.entries
}

func (r *Registry) Resolve(name string) (*ResourceDescriptor, bool) {
	if name == "" {
		return nil, false
	}
	desc, ok := r.byName[name]
	return desc, ok
}
