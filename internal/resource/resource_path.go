// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"iter"
	"slices"
	"strings"

	xslices "unikraft.com/cli/internal/x/slices"
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

func (fp FieldPath) Matches(spec FieldPath) bool {
	if len(fp) != len(spec) {
		return false
	}
	for i := range fp {
		if spec[i] != "*" && fp[i] != spec[i] {
			return false
		}
	}
	return true
}

// MatchesParent is like Matches, but also matches when spec refers to a parent
// path of fp (i.e. spec can be shorter than fp).
func (fp FieldPath) MatchesParent(spec FieldPath) bool {
	if len(fp) < len(spec) {
		return false
	}
	for i := range spec {
		if spec[i] != "*" && fp[i] != spec[i] {
			return false
		}
	}
	return true
}

func (fp FieldPath) MatchesString(s string) bool {
	return fp.Matches(ParseFieldPath(s))
}

func (fp FieldPath) String() string {
	return strings.Join(fp, ".")
}

// Leaf returns the last non-wildcard segment of the path.
// For example, "network.interfaces.*.ip" -> "ip".
func (fp FieldPath) Leaf() string {
	for i := len(fp) - 1; i >= 0; i-- {
		if fp[i] == "*" {
			continue
		}
		return fp[i]
	}
	return ""
}

// IterFields iterates over all fields and their subfields, yielding the full
// path to each field along with a pointer to the field itself.
func IterFields(fields []Field) iter.Seq2[FieldPath, *Field] {
	return func(yield func(FieldPath, *Field) bool) {
		iterFields(nil, fields, yield)
	}
}

func iterFields(path FieldPath, fields []Field, yield func(FieldPath, *Field) bool) bool {
	for i := range fields {
		field := &fields[i]
		path := append(slices.Clone(path), field.Name)
		if !yield(path, field) {
			return false
		}
		if !iterFields(path, field.Subfields, yield) {
			return false
		}
	}
	return true
}

// GetFieldByPath retrieves all fields matching the given FieldPath.
//
// It can be used similarly to FilterFieldsByPath, but instead of retaining the
// structure, it flattens the result to contain only the matching fields.
func GetFieldByPath(fields []Field, spec FieldPath) []Field {
	return getFieldByPath(nil, fields, spec)
}

// GetFieldByPathString is a convenience wrapper around GetFieldByPath that
// parses the path string first.
func GetFieldByPathString(fields []Field, path string) []Field {
	return GetFieldByPath(fields, ParseFieldPath(path))
}

func getFieldByPath(parent *Field, fields []Field, spec FieldPath) []Field {
	if len(spec) == 0 {
		return fields
	}

	result := make([]Field, 0)
	for _, field := range fields {
		if spec[0] == field.Name || spec[0] == "*" && parent != nil && parent.Elem != nil {
			if len(spec) == 1 {
				result = append(result, pruneFields(field))
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
	return field.Subfields, xslices.DedupeStringer(missing)
}

func filterFieldsByPath(field Field, specs []FieldPath, strict bool) (result Field, missing []FieldPath) {
	if len(specs) == 0 {
		if !strict {
			field = mergeElem(field)
		}
		field = pruneFields(field)
		return field, nil
	}

	result = field
	result.Value = nil
	result.Subfields = nil

	for len(specs) > 0 {
		// find the first "group" of specs with the same root
		target, rest := slices.Clone(specs[:1]), slices.Clone(specs[1:])
		specs = rest

		// check for an exact match
		if len(target[0]) == 0 {
			if field.Value != nil {
				// if we have a value, then include only that, don't include subfields
				// (we'll go grab those later)
				result.Value = field.Value
			} else {
				// if we don't, then recursively include all subfields
				if !strict && field.Elem != nil {
					elem := *field.Elem
					elem.Name = "*"
					elem = mergeElem(elem)
					elem = pruneFields(elem)
					result.Subfields = append(result.Subfields, elem)
				}
				for _, subfield := range field.Subfields {
					if !strict {
						subfield = mergeElem(subfield)
					}
					subfield = pruneFields(subfield)
					result.Subfields = append(result.Subfields, subfield)
				}
			}
			continue
		}

		spec := target[0][0]
		for i := range target {
			target[i] = target[i][1:]
		}

		// find all fields matching the root
		matched := false
		for _, subfield := range field.Subfields {
			if spec == subfield.Name || spec == "*" && field.Elem != nil {
				matched = true
				subfield, missed := filterFieldsByPath(subfield, target, strict)
				result.Subfields = append(result.Subfields, subfield)

				for _, miss := range missed {
					missing = append(missing, append(FieldPath{spec}, miss...))
				}
			}
		}

		// filter the elem to the root as well (if it exists)
		if !strict && field.Elem != nil {
			subfield, missed := filterFieldsByPath(*field.Elem, target, strict)
			if !strict {
				matched = true
				subfield.Name = spec // NOTE: override the name
				result.Subfields = append(result.Subfields, subfield)

				for _, miss := range missed {
					missing = append(missing, append(FieldPath{spec}, miss...))
				}
			}
		}

		if !matched {
			missing = append(missing, FieldPath{spec})
		}
	}

	return result, missing
}

func pruneFields(field Field) Field {
	if field.Elem != nil {
		elem := pruneFields(*field.Elem)
		field.Elem = &elem
	}
	if field.Value != nil {
		field.Subfields = nil
		return field
	}
	field.Subfields = slices.Clone(field.Subfields)
	for i := range field.Subfields {
		field.Subfields[i] = pruneFields(field.Subfields[i])
	}
	return field
}

func mergeElem(field Field) Field {
	if field.Elem != nil {
		elem := mergeElem(*field.Elem)
		elem.Name = "*"
		field.Elem = nil
		field.Subfields = append(field.Subfields, elem)
	}
	field.Subfields = slices.Clone(field.Subfields)
	for i := range field.Subfields {
		field.Subfields[i] = mergeElem(field.Subfields[i])
	}
	return field
}
