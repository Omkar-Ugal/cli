// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"fmt"
	"iter"
	"slices"
	"sort"
	"strings"

	"github.com/alecthomas/kong"

	"unikraft.com/cli/internal/cmd"
)

// CreateParser creates a kong parser for documentation generation.
func CreateParser() (*kong.Kong, error) {
	return cmd.NewParser(&cmd.UnikraftCLI{})
}

// NodePath returns the full command path for a node (e.g., "unikraft instance list").
func NodePath(node *kong.Node) string {
	if node.Parent == nil {
		return node.Name
	}
	return NodePath(node.Parent) + " " + node.Name
}

// IsRunnable returns true if a node represents a runnable command.
func IsRunnable(node *kong.Node) bool {
	return len(node.Positional) > 0 || len(node.Children) == 0 || node.Parent == nil
}

// IterChildren returns an iterator over all non-hidden children recursively, sorted by name.
func IterChildren(node *kong.Node) iter.Seq[*kong.Node] {
	return func(yield func(*kong.Node) bool) {
		if !yield(node) {
			return
		}
		children := make([]*kong.Node, 0, len(node.Children))
		for _, c := range node.Children {
			if c == nil || c.Hidden {
				continue
			}
			children = append(children, c)
		}

		for _, child := range slices.SortedFunc(slices.Values(children), func(a, b *kong.Node) int {
			return strings.Compare(a.Name, b.Name)
		}) {
			for desc := range IterChildren(child) {
				if !yield(desc) {
					return
				}
			}
		}
	}
}

// SeeAlso returns an iterator over related commands (parent and children).
func SeeAlso(node *kong.Node) iter.Seq[*kong.Node] {
	return func(yield func(*kong.Node) bool) {
		if node.Parent != nil {
			if !yield(node.Parent) {
				return
			}
		}
		for _, c := range node.Children {
			if c != nil && !c.Hidden {
				if !yield(c) {
					return
				}
			}
		}
	}
}

// CollectLocalFlags returns non-hidden flags defined on this node.
func CollectLocalFlags(node *kong.Node) []*kong.Flag {
	var flags []*kong.Flag
	for _, f := range node.Flags {
		if f.Hidden || f.Name == "help" {
			continue
		}
		flags = append(flags, f)
	}
	sort.Slice(flags, func(i, j int) bool {
		return flags[i].Name < flags[j].Name
	})
	return flags
}

// CollectInheritedFlags returns non-hidden flags from ancestor nodes.
func CollectInheritedFlags(node *kong.Node) []*kong.Flag {
	if node.Parent == nil {
		return nil
	}

	var flags []*kong.Flag
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		for _, f := range parent.Flags {
			if f.Hidden || f.Name == "help" {
				continue
			}
			flags = append(flags, f)
		}
	}

	sort.Slice(flags, func(i, j int) bool {
		return flags[i].Name < flags[j].Name
	})
	return flags
}

// FormatFlag formats a kong flag for display.
func FormatFlag(f *kong.Flag) string {
	var parts []string

	if f.Short != 0 {
		parts = append(parts, fmt.Sprintf("-%c", f.Short))
	}

	longFlag := "--" + f.Name
	if f.PlaceHolder != "" {
		longFlag += " " + f.PlaceHolder
	} else if !f.IsBool() {
		longFlag += " " + strings.ToUpper(f.Name)
	}
	parts = append(parts, longFlag)

	line := "  " + strings.Join(parts, ", ")

	if f.Help != "" {
		if len(line) < 30 {
			line += strings.Repeat(" ", 30-len(line))
		} else {
			line += "  "
		}
		line += f.Help
	}

	if f.Default != "" && f.Default != "false" && f.Default != "0" && f.Default != "[]" {
		line += fmt.Sprintf(" (default %s)", f.Default)
	}

	return line
}
