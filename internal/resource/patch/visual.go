// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package patch

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"slices"

	"sigs.k8s.io/yaml"
	"unikraft.com/x/log"

	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/resource"
)

func VisualEdit(ctx context.Context, cfg *config.Config, res resource.Resource, fields []resource.Field, patches []resource.Field) ([]resource.Field, error) {
	return visualEdit(ctx, cfg, res, fields, patches, false)
}

func VisualCreate(ctx context.Context, cfg *config.Config, res resource.Resource, fields []resource.Field, creates []resource.Field) ([]resource.Field, error) {
	return visualEdit(ctx, cfg, res, fields, creates, true)
}

// visualEdit opens an editor for the user to modify fields visually.
//
// It takes all the fields and already existing patched fields as input, and
// returns all patched fields after editing.
func visualEdit(ctx context.Context, cfg *config.Config, res resource.Resource, fields []resource.Field, patches []resource.Field, create bool) ([]resource.Field, error) {
	data, err := saveFields(res, fields, patches, create)
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

	cmd := exec.CommandContext(ctx, editor, tmpfile.Name())
	cmd.Stdin = cfg.Stdin
	cmd.Stdout = cfg.Stdout
	cmd.Stderr = cfg.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("editor exited with error: %w", err)
	}

	editedData, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read edited file: %w", err)
	}
	editedData = bytes.TrimSpace(editedData)
	if len(editedData) == 0 {
		return nil, fmt.Errorf("edited data is empty")
	}

	fields, err = loadFieldPatches(ctx, fields, editedData, create)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize edited fields: %w", err)
	}
	return fields, nil
}

func saveFields(res resource.Resource, fields []resource.Field, patches []resource.Field, create bool) ([]byte, error) {
	fields = resource.CloneFields(fields)

	patchMap := make(map[string]resource.Field)
	for key, field := range resource.IterFields(patches) {
		patchMap[key.String()] = *field
	}

	for key, field := range resource.IterFields(fields) {
		keyStr := key.String()

		var patch *resource.Patch
		if create {
			patch = field.Create
		} else {
			patch = field.Edit
		}
		if patch == nil || patch.Set == nil {
			continue
		}

		// verify that the patch set type is assignable to the value type
		value, err := collectValue(*field, reflect.TypeOf(patch.Set))
		if err != nil {
			return nil, fmt.Errorf("failed to collect value for field %s: %w", keyStr, err)
		}
		// TODO: instead of relying on patch.Set and field.Value being the same
		// type, we should have each Resource store the actual patch.Set content
		// (not just the type-info). Then patching should only update the
		// patch.Set, and never actually touch the Value.
		if !reflect.TypeOf(value).AssignableTo(reflect.TypeOf(patch.Set)) {
			return nil, fmt.Errorf("%s of value %T cannot be patched with %T", keyStr, value, patch.Set)
		}

		// write already set patches into fields
		if patchedField, ok := patchMap[keyStr]; ok {
			var patch *resource.Patch
			if create {
				patch = patchedField.Create
			} else {
				patch = patchedField.Edit
			}
			if patch == nil {
				return nil, fmt.Errorf("no patch available for field %s", keyStr)
			}
			if patch.Add != nil {
				return nil, fmt.Errorf("%s was added, but visual editing does not support this", keyStr)
			}
			if patch.Del != nil {
				return nil, fmt.Errorf("%s was deleted, but visual editing does not support this", keyStr)
			}
			if patch.Set == nil {
				return nil, fmt.Errorf("%s is not settable", keyStr)
			}
			err := storeValue(field, reflect.ValueOf(patch.Set))
			if err != nil {
				return nil, fmt.Errorf("failed to store patched value for field %s: %w", keyStr, err)
			}
		}
	}

	var patchableFields []resource.Field
	if create {
		patchableFields = FilterCreatableFields(fields)
	} else {
		patchableFields = FilterPatchableFields(fields)
	}
	result, err := yaml.Marshal(resource.FieldsToMap(patchableFields))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to YAML: %w", err)
	}

	result = append(fmt.Appendf(nil, "# %s %s\n", res.Type().Name, res.Key()), result...)

	return result, nil
}

func loadFieldPatches(ctx context.Context, fields []resource.Field, data []byte, create bool) ([]resource.Field, error) {
	fields = resource.CloneFields(fields)

	var obj map[string]any
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal edited data: %w", err)
	}

	patchedFields, missing, err := resource.MapToFields(fields, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to map edited data to fields: %w", err)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("unknown fields: %v", missing)
	}

	before := resource.Field{Subfields: fields}
	after := resource.Field{Subfields: patchedFields}
	err = patchField(resource.FieldPath{}, &before, &after, create)
	if err != nil {
		return nil, err
	}

	for fieldPath, field := range resource.IterFields(fields) {
		var patch *resource.Patch
		if create {
			patch = field.Create
		} else {
			patch = field.Edit
		}
		if patch == nil || patch.Set == nil {
			continue
		}
		value, err := collectValue(*field, reflect.TypeOf(patch.Set))
		if err != nil {
			return nil, fmt.Errorf("failed to collect value for field %s: %w", fieldPath.String(), err)
		}
		log.G(ctx).
			Debug().
			Str("field", fieldPath.String()).
			Any("old", value).
			Any("new", patch.Set).
			Msg("patched field")
	}

	if create {
		fields = FilterCreatableFields(fields)
	} else {
		fields = FilterPatchableFields(fields)
	}
	return fields, nil
}

// patchField applies the changes from after to before, modifying before in
// place by setting resource.Patch fields.
func patchField(path resource.FieldPath, before *resource.Field, after *resource.Field, create bool) error {
	var patch *resource.Patch
	if create {
		patch = before.Create
	} else {
		patch = before.Edit
	}

	if before.Name != after.Name {
		return fmt.Errorf("field name changed from %s to %s at %s", before.Name, after.Name, path.String())
	}

	if !reflect.DeepEqual(before.Value, after.Value) {
		if patch == nil {
			return fmt.Errorf("no patch available for field %s", path.String())
		}
		if patch.Set == nil {
			return fmt.Errorf("field %s is not settable", path.String())
		}
		if !reflect.TypeOf(after.Value).AssignableTo(reflect.TypeOf(patch.Set)) {
			return fmt.Errorf("cannot assign value of type %T to patch of type %T for field %s", after.Value, patch.Set, path.String())
		}
		patch.Set = after.Value
		patch.Add = nil
		patch.Del = nil
	} else if before.Elem != nil && !reflect.DeepEqual(before.Subfields, after.Subfields) {
		if patch == nil {
			return fmt.Errorf("no patch available for field %s", path.String())
		}
		if patch.Set == nil {
			return fmt.Errorf("field %s is not settable", path.String())
		}
		original, err := collectValue(*before, reflect.TypeOf(patch.Set))
		if err != nil {
			return fmt.Errorf("failed to collect value for field %s: %w", path.String(), err)
		}
		next, err := collectValue(*after, reflect.TypeOf(patch.Set))
		if err != nil {
			return fmt.Errorf("failed to collect value for field %s: %w", path.String(), err)
		}
		if !reflect.DeepEqual(original, next) {
			// due to weirdness in having saved-and-loaded the values, the subfields
			// might not be exactly identical, so we check DeepEqual on the collected
			// values as well
			patch.Set = next
			patch.Add = nil
			patch.Del = nil
		} else {
			patch = nil
		}
	} else {
		if len(before.Subfields) != len(after.Subfields) {
			return fmt.Errorf("number of subfields changed for field %s", path.String())
		}
		for i := range before.Subfields {
			before := &before.Subfields[i]
			after := &after.Subfields[i]
			path := append(slices.Clone(path), before.Name)
			err := patchField(path, before, after, create)
			if err != nil {
				return err
			}
		}
		patch = nil
	}

	if create {
		before.Create = patch
	} else {
		before.Edit = patch
	}

	return nil
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
