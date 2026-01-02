// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"sigs.k8s.io/yaml"
)

func VisualEdit(fields []Field, patches []Field) ([]Field, error) {
	return visualEdit(fields, patches, false)
}

func VisualCreate(fields []Field, creates []Field) ([]Field, error) {
	return visualEdit(fields, creates, true)
}

// visualEdit opens an editor for the user to modify fields visually.
//
// It takes all the fields and already existing patched fields as input, and
// returns all patched fields after editing.
func visualEdit(fields []Field, patches []Field, create bool) ([]Field, error) {
	data, err := saveFields(fields, patches, create)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize fields: %w", err)
	}

	tmpfile, err := os.CreateTemp("", "unikraft-edit-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(data); err != nil {
		tmpfile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	editor, err := getEditor()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("editor exited with error: %w", err)
	}

	editedData, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read edited file: %w", err)
	}

	fields, err = loadFieldPatches(fields, editedData, create)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize edited fields: %w", err)
	}
	return fields, nil
}

func saveFields(fields []Field, patches []Field, create bool) ([]byte, error) {
	patchMap := make(map[string]any)
	for key, field := range IterFields(patches) {
		keyStr := key.String()
		var patch *Patch
		if create {
			patch = field.Create
		} else {
			patch = field.Patch
		}
		if patch == nil {
			return nil, fmt.Errorf("cannot visual edit field %s with no patch", keyStr)
		}
		if patch.Add != nil {
			return nil, fmt.Errorf("cannot visual edit field %s with Add patch", keyStr)
		}
		if patch.Del != nil {
			return nil, fmt.Errorf("cannot visual edit field %s with Del patch", keyStr)
		}
		if patch.Set == nil {
			continue
		}

		if !reflect.TypeOf(patch.Set).AssignableTo(reflect.TypeOf(field.Value)) {
			return nil, fmt.Errorf("patch set type for field %s is not assignable to value type", field.Name)
		}
		patchMap[keyStr] = patch.Set
	}

	var patchableFields []Field
	if create {
		patchableFields = filterCreatableFields(fields)
	} else {
		patchableFields = filterPatchableFields(fields)
	}
	for key, field := range IterFields(patchableFields) {
		keyStr := key.String()
		if val, ok := patchMap[keyStr]; ok {
			field.Value = val
		}
	}
	obj := fieldsToMap(patchableFields)
	return yaml.Marshal(obj)
}

func loadFieldPatches(fields []Field, data []byte, create bool) ([]Field, error) {
	fields = CloneFields(fields)

	var obj map[string]any
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal edited data: %w", err)
	}

	for key, field := range IterFields(fields) {
		if create {
			patch := field.Create
			if patch == nil {
				continue
			}
			field.Create = nil
			if patch.Set == nil {
				continue
			}
		} else {
			patch := field.Patch
			if patch == nil {
				continue
			}
			field.Patch = nil
			if patch.Set == nil {
				continue
			}
		}

		result, ok := mapdig(obj, key...)
		if !ok {
			continue
		}
		newValue := reflect.New(reflect.TypeOf(field.Value))
		err := mapstructure.Decode(result, newValue.Interface())
		if err != nil {
			return nil, fmt.Errorf("failed to decode field %s: %w", key, err)
		}

		if reflect.DeepEqual(field.Value, newValue.Elem().Interface()) {
			continue
		}

		patch := Patch{Set: newValue.Elem().Interface()}
		if create {
			field.Create = &patch
		} else {
			field.Patch = &patch
		}
	}

	if create {
		return filterCreatableFields(fields), nil
	}
	return filterPatchableFields(fields), nil
}

func getEditor() (string, error) {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor, nil
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, nil
	}
	return "", fmt.Errorf("no editor set: please set $VISUAL or $EDITOR")
}

func mapdig(m map[string]any, keys ...string) (any, bool) {
	var current any = m

	for _, key := range keys {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = currentMap[key]
		if !ok {
			return nil, false
		}
	}

	return current, true
}
