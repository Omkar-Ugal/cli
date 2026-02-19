// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"context"

	"github.com/containerd/containerd/v2/pkg/filters"

	"unikraft.com/cli/internal/resource"
	xfilters "unikraft.com/cli/internal/x/filters"
)

type resolvedResource struct {
	resource.Resource
	fields []resource.Field
}

func (r resolvedResource) Fields() ([]resource.Field, error) {
	return resource.CloneFields(r.fields), nil
}

func resolveResources(ctx context.Context, resources []resource.Resource, filter filters.Filter, sortSpecs []SortSpec, fieldSpecs []string) ([]resource.Resource, error) {
	if len(resources) == 0 {
		return resources, nil
	}

	resolveAll := false
	paths := make([]resource.FieldPath, 0)
	if filter != nil {
		for _, key := range xfilters.Keys(filter) {
			paths = append(paths, resource.FieldPath(key))
		}
	}
	for _, spec := range sortSpecs {
		paths = append(paths, spec.Path)
	}
	selected, selectAll := fieldPathsFromSpecs(fieldSpecs)
	if selectAll {
		resolveAll = true
	} else {
		paths = append(paths, selected...)
	}

	if !resolveAll && len(paths) == 0 {
		return resources, nil
	}

	paths = dedupeFieldPaths(paths)

	resolved := make([]resource.Resource, len(resources))
	for i, res := range resources {
		resolvedRes, err := resolveResource(ctx, res, paths, resolveAll)
		if err != nil {
			return nil, err
		}
		resolved[i] = resolvedRes
	}
	return resolved, nil
}

func resolveResource(ctx context.Context, res resource.Resource, paths []resource.FieldPath, resolveAll bool) (resource.Resource, error) {
	fields, err := res.Fields()
	if err != nil {
		return nil, err
	}

	var resolved []resource.Field
	if resolveAll {
		resolved, err = resolveAllFields(ctx, fields)
	} else {
		resolved, err = resolveFields(ctx, fields, paths)
	}
	if err != nil {
		return nil, err
	}

	return resolvedResource{Resource: res, fields: resolved}, nil
}

func fieldPathsFromSpecs(specs []string) ([]resource.FieldPath, bool) {
	paths := make([]resource.FieldPath, 0, len(specs))
	resolveAll := false
	for _, spec := range specs {
		if spec == "" {
			continue
		}
		if spec == "all" {
			resolveAll = true
			continue
		}
		switch spec[0] {
		case '+', '-':
			spec = spec[1:]
		}
		if spec == "" {
			continue
		}
		paths = append(paths, resource.ParseFieldPath(spec))
	}
	return paths, resolveAll
}

func dedupeFieldPaths(paths []resource.FieldPath) []resource.FieldPath {
	if len(paths) <= 1 {
		return paths
	}
	seen := make(map[string]struct{}, len(paths))
	result := make([]resource.FieldPath, 0, len(paths))
	for _, path := range paths {
		key := path.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, path)
	}
	return result
}
