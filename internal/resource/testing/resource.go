// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package testing

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"unikraft.com/cli/internal/resource"
)

type TestResource struct {
	ID       string
	Name     string
	URL      string
	Settings TestSettings
	Authors  []TestAuthor
}

var (
	_ resource.Resource          = (*TestResource)(nil)
	_ resource.GettableResource  = (*TestResource)(nil)
	_ resource.EditableResource  = (*TestResource)(nil)
	_ resource.CreatableResource = (*TestResource)(nil)
	_ resource.DeletableResource = (*TestResource)(nil)
)

type TestSettings struct {
	X int
	Y string
}

type TestAuthor struct {
	Name  string
	Email string
}

var TestStore = map[string]TestResource{}

func (TestResource) Type() resource.Type {
	return resource.Type{
		Name:  "test",
		Names: "tests",
	}
}

func (t TestResource) Key() string {
	return t.Name
}

func (t TestResource) Raw() any {
	return t
}

func (t TestResource) Fields() ([]resource.Field, error) {
	return []resource.Field{
		{
			Name:      "id",
			Value:     t.ID,
			Verbosity: resource.FieldVerbosityShort,
		},
		{
			Name:      "name",
			Value:     t.Name,
			Verbosity: resource.FieldVerbosityShort,
			Create: &resource.Patch{
				Set: "",
			},
		},
		{
			Name:      "url",
			Value:     t.URL,
			Verbosity: resource.FieldVerbosityShort,
		},
		{
			Name:      "settings",
			Verbosity: resource.FieldVerbosityLong,
			Subfields: []resource.Field{
				{
					Name:      "x",
					Value:     t.Settings.X,
					Verbosity: resource.FieldVerbosityLong,
					Create: &resource.Patch{
						Set: 0,
					},
					Patch: &resource.Patch{
						Set: 0,
					},
				},
				{
					Name:      "y",
					Value:     t.Settings.Y,
					Verbosity: resource.FieldVerbosityLong,
					Create: &resource.Patch{
						Set: "",
					},
					Patch: &resource.Patch{
						Set: "",
					},
				},
			},
		},
		{
			Name:      "authors",
			Verbosity: resource.FieldVerbosityLong,
			Elem: &resource.Field{
				Subfields: []resource.Field{
					{
						Name:      "name",
						Verbosity: resource.FieldVerbosityLong,
					},
					{
						Name:      "email",
						Verbosity: resource.FieldVerbosityLong,
					},
				},
			},
			Subfields: func() []resource.Field {
				var fields []resource.Field
				for i, author := range t.Authors {
					fields = append(fields, resource.Field{
						Name:      fmt.Sprintf("%d", i),
						Verbosity: resource.FieldVerbosityLong,
						Subfields: []resource.Field{
							{
								Name:      "name",
								Value:     author.Name,
								Verbosity: resource.FieldVerbosityLong,
							},
							{
								Name:      "email",
								Value:     author.Email,
								Verbosity: resource.FieldVerbosityLong,
							},
						},
					})
				}
				return fields
			}(),
		},
	}, nil
}

func (TestResource) List(ctx context.Context) ([]resource.Resource, error) {
	var resources []resource.Resource
	for _, t := range TestStore {
		resources = append(resources, t)
	}
	// Sort by ID for deterministic output
	slices.SortFunc(resources, func(a, b resource.Resource) int {
		return strings.Compare(a.(TestResource).ID, b.(TestResource).ID)
	})
	return resources, nil
}

func (TestResource) Get(ctx context.Context, keys []string) ([]resource.Resource, error) {
	// Build a map for lookup
	resourceMap := make(map[string]TestResource)
	for _, key := range keys {
		if t, ok := TestStore[key]; ok {
			resourceMap[key] = t
		}
	}

	// Return resources in the order of keys provided
	var resources []resource.Resource
	for _, key := range keys {
		if t, ok := resourceMap[key]; ok {
			resources = append(resources, t)
		}
	}
	return resources, nil
}

func (TestResource) Create(ctx context.Context, fields []resource.Field) (resource.Resource, error) {
	t := TestResource{
		Settings: TestSettings{},
	}

	for key, field := range resource.IterFields(fields) {
		if field.Create == nil || field.Create.Set == nil {
			continue
		}
		switch key.String() {
		case "name":
			t.Name = field.Create.Set.(string)
		case "settings.x":
			t.Settings.X = field.Create.Set.(int)
		case "settings.y":
			t.Settings.Y = field.Create.Set.(string)
		}
	}

	TestStore[t.Name] = t
	return t, nil
}

func (TestResource) Edit(ctx context.Context, target resource.Resource, fields []resource.Field) (resource.Resource, error) {
	t := target.(TestResource)

	for key, field := range resource.IterFields(fields) {
		if field.Patch == nil || field.Patch.Set == nil {
			continue
		}
		switch key.String() {
		case "settings.x":
			t.Settings.X = field.Patch.Set.(int)
		case "settings.y":
			t.Settings.Y = field.Patch.Set.(string)
		}
	}

	TestStore[t.Name] = t
	return t, nil
}

func (TestResource) Delete(ctx context.Context, targets []resource.Resource) error {
	for _, target := range targets {
		t := target.(TestResource)
		delete(TestStore, t.Name)
	}
	return nil
}
