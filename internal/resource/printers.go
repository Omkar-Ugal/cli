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
	"slices"
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
		fields, err = resourceFields(fields, FieldVerbosityLong, specs)
		if err != nil {
			return err
		}

		if i > 0 {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
		}
		if err := printKVFields(out, fields, ""); err != nil {
			return err
		}
	}
	return nil
}

func printKVFields(out io.Writer, fields []Field, prefix string) error {
	for _, field := range fields {
		var line bytes.Buffer
		line.WriteString(prefix)

		line.WriteString(field.Name + ":")
		if field.Value != nil {
			line.WriteString(" ")
			line.WriteString(field.ValueString())
		}
		line.WriteString("\n")
		if _, err := io.Copy(out, &line); err != nil {
			return err
		}

		if err := printKVFields(out, field.Subfields, prefix+"  "); err != nil {
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
	headers, err = resourceFields(headers, FieldVerbosityShort, fieldSpecs)
	if err != nil {
		return err
	}
	headers = unpackFields(headers)
	headers = slices.DeleteFunc(headers, func(field Field) bool {
		return len(field.Subfields) > 0
	})

	for _, header := range headers {
		name := strings.ToUpper(header.Name)
		_, err := fmt.Fprintf(out, "%s\t", lipgloss.NewStyle().SetString(name).Bold(true).String())
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
		fieldsMap := make(map[string]*Field, len(fields))
		for _, field := range IterFields(fields) {
			fieldsMap[field.Name] = field
		}

		for i, header := range headers {
			field, ok := fieldsMap[header.Name]
			if !ok || field.Value == nil {
				_, err := fmt.Fprint(out, "\t")
				if err != nil {
					return err
				}
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
			_, err := fmt.Fprint(out, value)
			if err != nil {
				return err
			}
			if i < len(headers)-1 {
				_, err := fmt.Fprint(out, "\t")
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
		input = append(input, fieldValues(fields))
	}

	tmpl, err := template.New("out").Funcs(sprig.TxtFuncMap()).Parse(tmplStr)
	if err != nil {
		return err
	}
	return tmpl.Execute(out, input)
}

func resourceFields(fields []Field, verbosity FieldVerbosity, fieldSpecs []string) ([]Field, error) {
	base := make(map[string]struct{})
	include := make(map[string]struct{})
	exclude := make(map[string]struct{})
	found := make(map[string]bool)
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
			include[field] = struct{}{}
		case '-':
			field = field[1:]
			exclude[field] = struct{}{}
		default:
			base[field] = struct{}{}
		}
		found[field] = false
	}

	fields = filterFields(fields, func(field Field, path []string) bool {
		for i := range path {
			key := strings.Join(path[:i+1], ".")
			if _, ok := found[key]; ok {
				found[key] = true
			}
			if _, ok := exclude[key]; ok {
				return false
			}
			if _, ok := include[key]; ok {
				return true
			}
			if _, ok := base[key]; ok {
				return true
			}
		}
		if len(base) > 0 {
			return false
		}
		return field.Verbosity >= verbosity
	}, nil)

	var notFound []string
	for field, ok := range found {
		if !ok {
			notFound = append(notFound, field)
		}
	}
	if len(notFound) > 0 {
		return nil, fmt.Errorf("unknown fields: %v", notFound)
	}

	return fields, nil
}
