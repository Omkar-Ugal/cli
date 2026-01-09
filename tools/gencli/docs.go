// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"unikraft.com/x/kingkong"
)

// DocsCmd generates markdown documentation for the CLI.
type DocsCmd struct {
	Outdir string `arg:"" required:"" help:"Output directory for generated documentation."`
}

func (c *DocsCmd) Run() error {
	lipgloss.SetColorProfile(termenv.Ascii)

	if err := os.MkdirAll(c.Outdir, 0o775); err != nil {
		return fmt.Errorf("could not create parent directories: %w", err)
	}

	parser, err := CreateParser()
	if err != nil {
		return fmt.Errorf("could not create parser: %w", err)
	}

	for child := range IterChildren(parser.Model.Node) {
		if err := generateMarkdown(child, c.Outdir); err != nil {
			return err
		}
	}
	return nil
}

func generateMarkdown(node *kong.Node, dir string) error {
	buf := new(bytes.Buffer)
	name := NodePath(node)

	buf.WriteString("---\n")
	buf.WriteString("title: \"" + name + "\"\n")
	buf.WriteString("description: " + node.Help + "\n")
	buf.WriteString("---\n\n")

	if node.Detail != "" {
		if node.Parent == nil {
			buf.WriteString("```\n")
		}
		buf.WriteString(node.Detail + "\n\n")
		if node.Parent == nil {
			buf.WriteString("```\n")
		}
	} else if node.Help != "" {
		buf.WriteString(node.Help + "\n\n")
	}

	if IsRunnable(node) {
		fmt.Fprintf(buf, "```\n%s\n```\n\n", kingkong.Summary(node))
	}

	printDocsOptions(buf, node)

	hasSeeAlso := false
	basename := strings.ReplaceAll(name, " ", "/") + ".mdx"
	for related := range SeeAlso(node) {
		if !hasSeeAlso {
			buf.WriteString("## See Also\n\n")
			hasSeeAlso = true
		}

		relatedName := NodePath(related)
		link := strings.ReplaceAll(relatedName, " ", "/") + ".mdx"
		link, err := filepath.Rel(filepath.Dir(basename), link)
		if err != nil {
			return err
		}
		fmt.Fprintf(buf, "* [`%s`](%s): %s\n", relatedName, link, related.Help)
	}
	if hasSeeAlso {
		buf.WriteString("\n")
	}

	filename := filepath.Join(dir, basename)
	fmt.Printf("mkdir: %s\n", filepath.Dir(filename))

	if err := os.MkdirAll(filepath.Dir(filename), 0o775); err != nil {
		return err
	}

	fmt.Printf("write: %s\n", filename)

	w, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if w != nil {
			w.Close()
		}
	}()

	_, err = buf.WriteTo(w)
	if err != nil {
		return err
	}

	err = w.Close()
	w = nil
	return err
}

func printDocsOptions(buf *bytes.Buffer, node *kong.Node) {
	localFlags := CollectLocalFlags(node)
	if len(localFlags) > 0 {
		buf.WriteString("## Options\n\n```\n")
		for _, f := range localFlags {
			buf.WriteString(FormatFlag(f) + "\n")
		}
		buf.WriteString("```\n\n")
	}

	inheritedFlags := CollectInheritedFlags(node)
	if len(inheritedFlags) > 0 {
		buf.WriteString("## Options inherited from parent commands\n\n```\n")
		for _, f := range inheritedFlags {
			buf.WriteString(FormatFlag(f) + "\n")
		}
		buf.WriteString("```\n\n")
	}
}
