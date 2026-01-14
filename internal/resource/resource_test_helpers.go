// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

type Test struct {
	ID       string
	Name     string
	URL      string
	Settings TestSettings
	Authors  []TestAuthor
}

var (
	_ Resource          = (*Test)(nil)
	_ GettableResource  = (*Test)(nil)
	_ EditableResource  = (*Test)(nil)
	_ CreatableResource = (*Test)(nil)
	_ DeletableResource = (*Test)(nil)
)

type TestSettings struct {
	X int
	Y string
}

type TestAuthor struct {
	Name  string
	Email string
}

var TestStore = map[string]Test{}

func (Test) Type() Type {
	return Type{
		Name:  "test",
		Names: "tests",
	}
}

func (t Test) Key() string {
	return t.Name
}

func (t Test) Raw() any {
	return t
}

func (t Test) Fields() ([]Field, error) {
	return []Field{
		{
			Name:      "id",
			Value:     t.ID,
			Verbosity: FieldVerbosityShort,
		},
		{
			Name:      "name",
			Value:     t.Name,
			Verbosity: FieldVerbosityShort,
			Create: &Patch{
				Set: "",
			},
		},
		{
			Name:      "url",
			Value:     t.URL,
			Verbosity: FieldVerbosityShort,
		},
		{
			Name:      "settings",
			Verbosity: FieldVerbosityLong,
			Subfields: []Field{
				{
					Name:      "x",
					Value:     t.Settings.X,
					Verbosity: FieldVerbosityLong,
					Create: &Patch{
						Set: 0,
					},
					Patch: &Patch{
						Set: 0,
					},
				},
				{
					Name:      "y",
					Value:     t.Settings.Y,
					Verbosity: FieldVerbosityLong,
					Create: &Patch{
						Set: "",
					},
					Patch: &Patch{
						Set: "",
					},
				},
			},
		},
		{
			Name:      "authors",
			Verbosity: FieldVerbosityLong,
			Elem: &Field{
				Subfields: []Field{
					{
						Name:      "name",
						Verbosity: FieldVerbosityLong,
					},
					{
						Name:      "email",
						Verbosity: FieldVerbosityLong,
					},
				},
			},
			Subfields: func() []Field {
				var fields []Field
				for i, author := range t.Authors {
					fields = append(fields, Field{
						Name:      fmt.Sprintf("%d", i),
						Verbosity: FieldVerbosityLong,
						Subfields: []Field{
							{
								Name:      "name",
								Value:     author.Name,
								Verbosity: FieldVerbosityLong,
							},
							{
								Name:      "email",
								Value:     author.Email,
								Verbosity: FieldVerbosityLong,
							},
						},
					})
				}
				return fields
			}(),
		},
	}, nil
}

func (Test) List(ctx context.Context) ([]Resource, error) {
	var resources []Resource
	for _, t := range TestStore {
		resources = append(resources, t)
	}
	// Sort by ID for deterministic output
	slices.SortFunc(resources, func(a, b Resource) int {
		return strings.Compare(a.(Test).ID, b.(Test).ID)
	})
	return resources, nil
}

func (Test) Get(ctx context.Context, keys []string) ([]Resource, error) {
	// Build a map for lookup
	resourceMap := make(map[string]Test)
	for _, key := range keys {
		if t, ok := TestStore[key]; ok {
			resourceMap[key] = t
		}
	}

	// Return resources in the order of keys provided
	var resources []Resource
	for _, key := range keys {
		if t, ok := resourceMap[key]; ok {
			resources = append(resources, t)
		}
	}
	return resources, nil
}

func (Test) Create(ctx context.Context, fields []Field) (Resource, error) {
	t := Test{
		Settings: TestSettings{},
	}

	for key, field := range IterFields(fields) {
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

func (Test) Edit(ctx context.Context, target Resource, fields []Field) (Resource, error) {
	t := target.(Test)

	for key, field := range IterFields(fields) {
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

func (Test) Delete(ctx context.Context, targets []Resource) error {
	for _, target := range targets {
		t := target.(Test)
		delete(TestStore, t.Name)
	}
	return nil
}
