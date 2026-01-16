// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"slices"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/juju/ansiterm"
	"github.com/muesli/termenv"
	"sigs.k8s.io/yaml"

	"unikraft.com/cli/internal/kvwriter"
	"unikraft.com/cli/internal/resource"
	xslices "unikraft.com/cli/internal/x/slices"
)

type PrinterType string

const (
	PrinterTypeTable    PrinterType = "table"
	PrinterTypeKeyValue PrinterType = "kv"
	PrinterTypeJSON     PrinterType = "json"
	PrinterTypeYAML     PrinterType = "yaml"
	PrinterTypeRaw      PrinterType = "raw"
	PrinterTypeQuiet    PrinterType = "quiet"
	PrinterTypeTemplate PrinterType = "template"
)

func (pt PrinterType) Validate() error {
	switch pt {
	case PrinterTypeTable, PrinterTypeKeyValue, PrinterTypeJSON, PrinterTypeYAML, PrinterTypeRaw, PrinterTypeQuiet, PrinterTypeTemplate:
		return nil
	default:
		return fmt.Errorf("unknown printer type: %s", pt)
	}
}

type Printer struct {
	Type  PrinterType
	Value string
}

var _ encoding.TextUnmarshaler = (*Printer)(nil)

func (p *Printer) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}

	k, v, _ := strings.Cut(string(text), "=")
	pt := PrinterType(k)
	if err := pt.Validate(); err != nil {
		return err
	}
	p.Type = pt
	p.Value = v
	return nil
}

func ParsePrinter(s string) (Printer, error) {
	pr := Printer{}
	err := pr.UnmarshalText([]byte(s))
	if err != nil {
		return Printer{}, err
	}
	return pr, nil
}

func (p Printer) WithDefault(tp PrinterType) Printer {
	if p.Type == "" {
		p.Type = tp
	}
	return p
}

func (p Printer) Print(out io.Writer, fieldSpecs []string, base resource.Resource, resources ...resource.Resource) error {
	switch p.Type {
	case "":
		return fmt.Errorf("printer type not specified")
	case PrinterTypeTable:
		return printTableFormatted(out, fieldSpecs, base, resources...)
	case PrinterTypeKeyValue:
		return printKVFormatted(out, fieldSpecs, resources...)
	case PrinterTypeJSON:
		return printJSON(out, resources...)
	case PrinterTypeYAML:
		return printYAML(out, resources...)
	case PrinterTypeRaw:
		return printRaw(out, resources...)
	case PrinterTypeQuiet:
		return printQuiet(out, resources...)
	case PrinterTypeTemplate:
		return printTemplate(out, p.Value, resources...)
	default:
		return fmt.Errorf("unknown printer type: %s", p.Type)
	}
}

func printTableFormatted(out io.Writer, fieldSpecs []string, base resource.Resource, resources ...resource.Resource) error {
	tw := ansiterm.NewTabWriter(out, 0, 8, 2, ' ', 0)
	err := printTable(tw, fieldSpecs, base, resources...)
	if err != nil {
		return err
	}
	return tw.Flush()
}

func printKVFormatted(out io.Writer, fieldSpecs []string, resources ...resource.Resource) error {
	bw := kvwriter.KeyValueWriter(out)
	err := printKV(bw, fieldSpecs, resources...)
	if err != nil {
		return err
	}
	return bw.Flush()
}

func printKV(out io.Writer, specs []string, resources ...resource.Resource) error {
	for i, res := range resources {
		fields, err := res.Fields()
		if err != nil {
			return err
		}
		fields, err = resourceFields(fields, false, resource.FieldVerbosityLong, specs)
		if err != nil {
			return err
		}
		fields = resource.DedupeFields(fields)

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

func printKVFields(out io.Writer, parent *resource.Field, fields []resource.Field, current int, indent int) error {
	for i, field := range fields {
		var line bytes.Buffer
		nextCurrent := 0
		nextIndent := indent + 1
		if parent != nil && parent.Elem != nil {
			line.WriteString(strings.Repeat("  ", max(0, indent-1)))
			line.WriteString("- ")
			nextCurrent = indent
			nextIndent = indent
			if field.Value != nil {
				line.WriteString(field.FormatString())
				line.WriteString("\n")
			}
		} else {
			if i == 0 {
				line.WriteString(strings.Repeat("  ", max(0, indent-current)))
			} else {
				line.WriteString(strings.Repeat("  ", indent))
			}
			line.WriteString(field.Name + ":")
			if field.Value != nil {
				line.WriteString(" ")
				line.WriteString(field.FormatString())
			}
			line.WriteString("\n")
		}
		if _, err := io.Copy(out, &line); err != nil {
			return err
		}

		if field.Value == nil {
			if err := printKVFields(out, &field, field.Subfields, nextCurrent, nextIndent); err != nil {
				return err
			}
		}
	}
	return nil
}

func printTable(out io.Writer, fieldSpecs []string, base resource.Resource, resources ...resource.Resource) error {
	headers, err := base.Fields()
	if err != nil {
		return err
	}

	headers, err = resourceFields(headers, true, resource.FieldVerbosityShort, fieldSpecs)
	if err != nil {
		return err
	}

	headerPaths, headerFields := xslices.Collect2(resource.IterFields(headers))

	for i, header := range headerFields {
		if header.HasChildren() && header.Value == nil {
			continue
		}
		path := headerPaths[i]

		if i > 0 {
			_, err := fmt.Fprint(out, "\t")
			if err != nil {
				return err
			}
		}
		name := strings.ToUpper(headerName(path))
		_, err := fmt.Fprintf(out, "%s", lipgloss.NewStyle().SetString(name).Bold(true).String())
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(out)
	if err != nil {
		return err
	}

	for _, res := range resources {
		fields, err := res.Fields()
		if err != nil {
			return err
		}

		for i, header := range headerFields {
			if header.HasChildren() && header.Value == nil {
				continue
			}
			path := headerPaths[i]

			if i > 0 {
				_, err = fmt.Fprint(out, "\t")
				if err != nil {
					return err
				}
			}

			fields := resource.GetFieldByPath(fields, path)
			fieldIdx := -1
			for _, field := range resource.IterFields(fields) {
				if field.Value == nil {
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

				value := field.FormatString()
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

func headerName(path resource.FieldPath) string {
	for _, part := range slices.Backward(path) {
		if part == "*" {
			continue
		}
		return part
	}
	return ""
}

func printQuiet(out io.Writer, resources ...resource.Resource) error {
	for _, res := range resources {
		_, err := fmt.Fprintln(out, res.Key())
		if err != nil {
			return err
		}
	}
	return nil
}

func printJSON(out io.Writer, resources ...resource.Resource) error {
	input := make([]any, 0, len(resources))
	for _, res := range resources {
		fields, err := res.Fields()
		if err != nil {
			return err
		}
		input = append(input, resource.FieldsToMap(fields))
	}
	dt, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(dt))
	return err
}

func printYAML(out io.Writer, resources ...resource.Resource) error {
	input := make([]any, 0, len(resources))
	for _, res := range resources {
		fields, err := res.Fields()
		if err != nil {
			return err
		}
		input = append(input, resource.FieldsToMap(fields))
	}
	dt, err := yaml.Marshal(input)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(out, string(dt))
	return err
}

func printRaw(out io.Writer, resources ...resource.Resource) error {
	input := make([]any, 0, len(resources))
	for _, res := range resources {
		input = append(input, res.Raw())
	}
	dt, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(dt))
	return err
}

func printTemplate(out io.Writer, tmplStr string, resources ...resource.Resource) error {
	input := make([]any, 0, len(resources))
	for _, res := range resources {
		fields, err := res.Fields()
		if err != nil {
			return err
		}
		input = append(input, resource.FieldsToMap(fields))
	}

	tmpl, err := template.New("out").Funcs(sprig.TxtFuncMap()).Parse(tmplStr)
	if err != nil {
		return err
	}
	return tmpl.Execute(out, input)
}

func resourceFields(fields []resource.Field, header bool, verbosity resource.FieldVerbosity, fieldSpecs []string) ([]resource.Field, error) {
	var base []resource.FieldPath
	var include []resource.FieldPath
	var exclude []resource.FieldPath
	for _, field := range fieldSpecs {
		if len(field) == 0 {
			continue
		}
		if field == "all" {
			verbosity = resource.FieldVerbosityHidden
			if base == nil {
				base = []resource.FieldPath{}
			}
			continue
		}
		switch field[0] {
		case '+':
			field = field[1:]
			include = append(include, resource.ParseFieldPath(field))
		case '-':
			field = field[1:]
			exclude = append(exclude, resource.ParseFieldPath(field))
		default:
			base = append(base, resource.ParseFieldPath(field))
		}
	}

	var missing []resource.FieldPath

	result, missing := resource.FilterFieldsByPath(fields, base, !header)
	if base == nil {
		result = resource.FilterFields(result, func(field resource.Field) resource.FilterResult {
			if field.Verbosity < verbosity {
				return resource.FilterExclude
			}
			if !header && field.IsEmpty() {
				return resource.FilterExclude
			}
			return resource.FilterRecurse
		})
	}

	if len(include) > 0 {
		included, includeMissing := resource.FilterFieldsByPath(fields, include, !header)
		result = resource.MergeFields(result, included)
		missing = append(missing, includeMissing...)
	}

	if len(exclude) > 0 {
		excluded, excludeMissing := resource.FilterFieldsByPath(fields, exclude, !header)
		result = resource.RemoveFields(result, excluded)
		missing = append(missing, excludeMissing...)
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("unknown fields: %v", missing)
	}

	return result, nil
}

func printPatches(out io.Writer, fields []resource.Field, create bool) error {
	tw := kvwriter.KeyValueWriter(
		out,
		kvwriter.WithSeparator(":=", "+=", "-="),
		kvwriter.WithAlignedSeparator(),
	)
	for path, field := range resource.IterFields(fields) {
		var patch *resource.Patch
		if create {
			patch = field.Create
		} else {
			patch = field.Edit
		}
		if patch == nil {
			continue
		}
		if patch.Set != nil {
			if _, err := fmt.Fprintf(tw, "%s := %s\n", path.String(), resource.FormatValue(patch.Set)); err != nil {
				return err
			}
		}
		if patch.Add != nil {
			if _, err := fmt.Fprintf(tw, "%s += %s\n", path.String(), resource.FormatValue(patch.Add)); err != nil {
				return err
			}
		}
		if patch.Del != nil {
			if _, err := fmt.Fprintf(tw, "%s -= %s\n", path.String(), resource.FormatValue(patch.Del)); err != nil {
				return err
			}
		}
	}
	return tw.Flush()
}
