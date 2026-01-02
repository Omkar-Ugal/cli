// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"iter"
	"slices"
	"strings"
)

// FieldPath represents a dot-separated path to a field in a resource.
// For example, "network.interfaces.0.ip_address". It can also contain
// wildcards, for example, "network.interfaces.*.ip_address".
type FieldPath []string

func ParseFieldPath(s string) FieldPath {
	// NOTE: be careful if modifying this - since this same syntax is used by
	// containerd filters (for the --filter flag)
	parts := strings.Split(s, ".")
	return FieldPath(parts)
}

func (fp FieldPath) String() string {
	return strings.Join(fp, ".")
}

// IterFields iterates over all fields and their subfields, yielding the full
// path to each field along with a pointer to the field itself.
func IterFields(fields []Field) iter.Seq2[FieldPath, *Field] {
	return func(yield func(FieldPath, *Field) bool) {
		iterFields(nil, fields, yield)
	}
}

func iterFields(path FieldPath, fields []Field, yield func(FieldPath, *Field) bool) {
	for i := range fields {
		field := &fields[i]
		path := append(slices.Clone(path), field.Name)
		if !yield(path, field) {
			return
		}
		iterFields(path, field.Subfields, yield)
	}
}

// GetFieldByPath retrieves all fields matching the given FieldPath.
func GetFieldByPath(fields []Field, spec FieldPath) []Field {
	return getFieldByPath(nil, fields, spec)
}

func getFieldByPath(parent *Field, fields []Field, spec FieldPath) []Field {
	if len(spec) == 0 {
		return fields
	}

	result := make([]Field, 0)
	for _, field := range fields {
		if spec[0] == field.Name || spec[0] == "*" && parent != nil && parent.Elem != nil {
			if len(spec) == 1 {
				result = append(result, field)
			} else {
				subfields := getFieldByPath(&field, field.Subfields, spec[1:])
				result = append(result, subfields...)
			}
		}
	}
	return result
}

// FilterFieldsByPath filters the given fields based on the provided
// FieldPaths. It retains the field hierarchy, but re-orders the fields to
// match the provided specs.
//
// If strict is true, then the fields must exist - otherwise, we allow using
// the element type to match fields that may not exist yet.
func FilterFieldsByPath(fields []Field, specs []FieldPath, strict bool) ([]Field, []FieldPath) {
	field, missing := filterFieldsByPath(Field{
		Subfields: fields,
	}, specs, strict)
	return field.Subfields, missing
}

func filterFieldsByPath(field Field, specs []FieldPath, strict bool) (result Field, missing []FieldPath) {
	if len(specs) == 0 {
		return field, nil
	}

	// make a map to track found keys (to avoid duplicates)
	foundKeys := make(map[string]int)

	// remove all the fully matched specs
	originalLen := len(specs)
	specs = slices.Clone(specs)
	specs = slices.DeleteFunc(specs, func(fp FieldPath) bool {
		return len(fp) == 0
	})
	if len(specs) == 0 {
		return field, nil
	}
	if len(specs) < originalLen {
		for _, field := range field.Subfields {
			if _, ok := foundKeys[field.Name]; ok {
				continue
			}
			foundKeys[field.Name] = len(result.Subfields)
			result.Subfields = append(result.Subfields, field)
		}

		// keep going - we want to find missing keys
	}

	for len(specs) > 0 {
		// find the first "group" of specs with the same root
		target, rest := specs[:1], specs[1:]
		for i := 0; i < len(rest); {
			if len(rest[i]) > 0 && len(target[0]) > 0 && rest[i][0] == target[0][0] {
				target = append(target, rest[i])
				rest = append(rest[:i], rest[i+1:]...)
			} else {
				i++
			}
		}

		spec := target[0][0]
		for i := range target {
			target[i] = target[i][1:]
		}
		specs = rest

		// then find all fields matching the root of that group
		matched := false
		for _, subfield := range field.Subfields {
			if spec == subfield.Name || spec == "*" && field.Elem != nil {
				matched = true
				filtered, missed := filterFieldsByPath(subfield, target, strict)

				if idx, ok := foundKeys[subfield.Name]; ok {
					field := &result.Subfields[idx]
					mergeTopLevelField(field, &filtered)
				} else {
					subfield.Subfields = filtered.Subfields
					foundKeys[subfield.Name] = len(result.Subfields)
					result.Subfields = append(result.Subfields, subfield)
				}

				for _, miss := range missed {
					missing = append(missing, append(FieldPath{spec}, miss...))
				}
			}
		}
		if matched {
			continue
		}

		// try matching against the element type
		if !strict && field.Elem != nil {
			filtered, missed := filterFieldsByPath(*field.Elem, target, strict)
			if idx, ok := foundKeys[field.Elem.Name]; ok {
				field := &result.Subfields[idx]
				mergeTopLevelField(field, &filtered)
			} else {
				subfield := *field.Elem
				subfield.Name = spec
				subfield.Subfields = filtered.Subfields
				foundKeys[field.Elem.Name] = len(result.Subfields)
				result.Subfields = append(result.Subfields, subfield)
			}

			for _, miss := range missed {
				missing = append(missing, append(FieldPath{spec}, miss...))
			}

			continue
		}

		missing = append(missing, FieldPath{spec})
	}

	field.Subfields = result.Subfields
	return field, missing
}

func mergeTopLevelField(dest *Field, src *Field) {
	dest.Subfields = append(dest.Subfields, slices.DeleteFunc(slices.Clone(src.Subfields), func(f Field) bool {
		for _, existing := range dest.Subfields {
			if f.Name == existing.Name {
				return true
			}
		}
		return false
	})...)
}

func mergeFieldElems(field Field) Field {
	field.Subfields = slices.Clone(field.Subfields)
	if field.Elem != nil && len(field.Subfields) == 0 {
		elem := *field.Elem
		elem.Name = "*"
		field.Subfields = append(field.Subfields, elem)
		field.Elem = nil
	}
	for i := range field.Subfields {
		field.Subfields[i] = mergeFieldElems(field.Subfields[i])
	}
	return field
}
