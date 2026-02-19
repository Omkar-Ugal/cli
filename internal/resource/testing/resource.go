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
	ID        string
	Name      string
	State     string
	URL       string
	Hidden    string
	Invisible string
	Lazy      string // Computed via callback when requested
	Settings  TestSettings
	Authors   []TestAuthor
}

// CallbackInvocations tracks how many times callbacks were invoked.
var CallbackInvocations int

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

type Hooks struct {
	List   func(context.Context, func(context.Context) ([]resource.Resource, error)) ([]resource.Resource, error)
	Get    func(context.Context, []string, func(context.Context, []string) ([]resource.Resource, error)) ([]resource.Resource, error)
	Create func(context.Context, []resource.Field, func(context.Context, []resource.Field) ([]resource.Resource, error)) ([]resource.Resource, error)
	Delete func(context.Context, []resource.Resource, func(context.Context, []resource.Resource) error) error
}

var TestStore = map[string]TestResource{}

// TestHooks allows tests to override TestResource operations.
//
// This is intentionally package-level and mutable to keep tests lightweight;
// tests should reset it via t.Cleanup.
var TestHooks Hooks

func (TestResource) Type() resource.Type {
	return resource.Type{
		Name:  "test",
		Names: "tests",
	}
}

type staticKey string

func (k staticKey) String() string {
	return string(k)
}

func (t TestResource) Key() resource.Key {
	return staticKey(t.Name)
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
			Name:      "state",
			Value:     t.State,
			Verbosity: resource.FieldVerbosityShort,
		},
		{
			Name:      "url",
			Value:     t.URL,
			Verbosity: resource.FieldVerbosityShort,
		},
		{
			Name:      "hidden",
			Value:     t.Hidden,
			Verbosity: resource.FieldVerbosityHidden,
		},
		{
			Name:      "invisible",
			Value:     t.Invisible,
			Verbosity: resource.FieldVerbosityInvisible,
		},
		{
			Name:      "lazy",
			Value:     t.Lazy,
			Verbosity: resource.FieldVerbosityHidden,
			ValueCallback: func(ctx context.Context) (any, error) {
				CallbackInvocations++
				return "computed-" + t.Name, nil
			},
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
					Edit: &resource.Patch{
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
					Edit: &resource.Patch{
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
	original := func(context.Context) ([]resource.Resource, error) {
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
	if TestHooks.List != nil {
		return TestHooks.List(ctx, original)
	}
	return original(ctx)
}

func (TestResource) Get(ctx context.Context, keys []string) ([]resource.Resource, error) {
	original := func(_ context.Context, keys []string) ([]resource.Resource, error) {
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
	if TestHooks.Get != nil {
		return TestHooks.Get(ctx, keys, original)
	}
	return original(ctx, keys)
}

func (TestResource) Create(ctx context.Context, fields []resource.Field) ([]resource.Resource, error) {
	original := func(_ context.Context, fields []resource.Field) ([]resource.Resource, error) {
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
		return []resource.Resource{t}, nil
	}
	if TestHooks.Create != nil {
		return TestHooks.Create(ctx, fields, original)
	}
	return original(ctx, fields)
}

func (TestResource) Edit(ctx context.Context, target resource.Resource, fields []resource.Field) (resource.Resource, error) {
	t := target.(TestResource)

	for key, field := range resource.IterFields(fields) {
		if field.Edit == nil || field.Edit.Set == nil {
			continue
		}
		switch key.String() {
		case "settings.x":
			t.Settings.X = field.Edit.Set.(int)
		case "settings.y":
			t.Settings.Y = field.Edit.Set.(string)
		}
	}

	TestStore[t.Name] = t
	return t, nil
}

func (TestResource) Delete(ctx context.Context, targets []resource.Resource) error {
	original := func(_ context.Context, targets []resource.Resource) error {
		for _, target := range targets {
			t := target.(TestResource)
			delete(TestStore, t.Name)
		}
		return nil
	}
	if TestHooks.Delete != nil {
		return TestHooks.Delete(ctx, targets, original)
	}
	return original(ctx, targets)
}
