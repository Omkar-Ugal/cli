// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/juju/ansiterm"
	"github.com/muesli/termenv"
	"unikraft.com/cli/internal/kvwriter"
)

func printTable[R Resource](out io.Writer, fieldSpecs []string, resources ...Resource) error {
	tw := ansiterm.NewTabWriter(out, 0, 8, 2, ' ', 0)
	err := printTabSeparated[R](tw, fieldSpecs, resources...)
	if err != nil {
		return err
	}
	return tw.Flush()
}

func printInspect(out io.Writer, fieldSpecs []string, resources ...Resource) error {
	bw := kvwriter.KeyValueWriter(out, "")
	err := printKV(bw, fieldSpecs, resources...)
	if err != nil {
		return err
	}
	return bw.Flush()
}

func printKV(out io.Writer, specs []string, resources ...Resource) error {
	for i, resource := range resources {
		fields, err := resource.Fields()
		if err != nil {
			return err
		}
		fields, err = resourceFields(fields, false, FieldVerbosityLong, specs)
		if err != nil {
			return err
		}

		if i > 0 {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
		}
		if err := printKVFields(out, nil, fields, 0, 0); err != nil {
			return err
		}
	}
	return nil
}

func printKVFields(out io.Writer, parent *Field, fields []Field, current int, indent int) error {
	for i, field := range fields {
		var line bytes.Buffer
		nextCurrent := 0
		nextIndent := indent + 1
		if parent != nil && parent.Elem != nil {
			line.WriteString(strings.Repeat("  ", max(0, indent-1)))
			line.WriteString("- ")
			nextCurrent = indent
			nextIndent = indent
		} else {
			if i == 0 {
				line.WriteString(strings.Repeat("  ", max(0, indent-current)))
			} else {
				line.WriteString(strings.Repeat("  ", indent))
			}
			line.WriteString(field.Name + ":")
			if field.Value != nil {
				line.WriteString(" ")
				line.WriteString(field.ValueString())
			}
			line.WriteString("\n")
		}
		if _, err := io.Copy(out, &line); err != nil {
			return err
		}

		if err := printKVFields(out, &field, field.Subfields, nextCurrent, nextIndent); err != nil {
			return err
		}
	}
	return nil
}

func printTabSeparated[R Resource](out io.Writer, fieldSpecs []string, resources ...Resource) error {
	var r R
	headers, err := r.Fields()
	if err != nil {
		return err
	}

	headers, err = resourceFields(headers, true, FieldVerbosityShort, fieldSpecs)
	if err != nil {
		return err
	}

	headerIdx := -1
	for _, header := range IterFields(headers) {
		if header.HasChildren() {
			continue
		}
		headerIdx++

		if headerIdx > 0 {
			_, err := fmt.Fprint(out, "\t")
			if err != nil {
				return err
			}
		}
		name := strings.ToUpper(header.Name)
		_, err := fmt.Fprintf(out, "%s", lipgloss.NewStyle().SetString(name).Bold(true).String())
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(out)
	if err != nil {
		return err
	}

	for _, resource := range resources {
		fields, err := resource.Fields()
		if err != nil {
			return err
		}

		headerIdx := -1
		for path, header := range IterFields(headers) {
			if header.HasChildren() {
				continue
			}
			headerIdx++

			if headerIdx > 0 {
				_, err = fmt.Fprint(out, "\t")
				if err != nil {
					return err
				}
			}

			fields := GetFieldByPath(fields, path)
			fieldIdx := -1
			for _, field := range IterFields(fields) {
				if field.HasChildren() {
					continue
				}
				fieldIdx++

				if fieldIdx > 0 {
					_, err := fmt.Fprint(out, ", ")
					if err != nil {
						return err
					}
				}

				if field.Value == nil {
					continue
				}

				value := field.ValueString()
				if field.Hyperlink != "" {
					// TODO: use lipgloss styles when it supports hyperlinks
					// https://github.com/charmbracelet/lipgloss/issues/220
					if lipgloss.ColorProfile() != termenv.Ascii {
						value = ansi.SetHyperlink(field.Hyperlink) + value + ansi.ResetHyperlink()
					}
				}
				_, err = fmt.Fprint(out, value)
				if err != nil {
					return err
				}
			}
		}
		_, err = fmt.Fprintln(out)
		if err != nil {
			return err
		}
	}

	return nil
}

func printQuiet(out io.Writer, resources ...Resource) error {
	for _, resource := range resources {
		_, err := fmt.Fprintln(out, resource.Key())
		if err != nil {
			return err
		}
	}
	return nil
}

func printRaw(out io.Writer, resources ...Resource) error {
	input := make([]any, 0, len(resources))
	for _, resource := range resources {
		input = append(input, resource.Raw())
	}
	dt, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(dt))
	return err
}

func printTemplate(out io.Writer, tmplStr string, resources ...Resource) error {
	input := make([]any, 0, len(resources))
	for _, resource := range resources {
		fields, err := resource.Fields()
		if err != nil {
			return err
		}
		input = append(input, fieldsToMap(fields))
	}

	tmpl, err := template.New("out").Funcs(sprig.TxtFuncMap()).Parse(tmplStr)
	if err != nil {
		return err
	}
	return tmpl.Execute(out, input)
}

func resourceFields(fields []Field, header bool, verbosity FieldVerbosity, fieldSpecs []string) ([]Field, error) {
	var base []FieldPath
	var include []FieldPath
	var exclude []FieldPath
	for _, field := range fieldSpecs {
		if len(field) == 0 {
			continue
		}
		if field == "all" {
			verbosity = FieldVerbosityHidden
			continue
		}
		switch field[0] {
		case '+':
			field = field[1:]
			include = append(include, ParseFieldPath(field))
		case '-':
			field = field[1:]
			exclude = append(exclude, ParseFieldPath(field))
		default:
			base = append(base, ParseFieldPath(field))
		}
	}

	var missing []FieldPath

	result, missing := FilterFieldsByPath(fields, base, !header)
	if len(base) == 0 {
		result = filterFields(result, func(field Field) bool {
			return field.Verbosity >= verbosity
		})
	}

	if len(include) > 0 {
		included, includeMissing := FilterFieldsByPath(fields, include, !header)
		result = MergeFields(result, included)
		missing = append(missing, includeMissing...)
	}

	if len(exclude) > 0 {
		excluded, excludeMissing := FilterFieldsByPath(fields, exclude, !header)
		result = RemoveFields(result, excluded)
		missing = append(missing, excludeMissing...)
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("unknown fields: %v", missing)
	}

	if header {
		for i := range result {
			result[i] = mergeFieldElems(result[i])
		}
	}

	return result, nil
}
