// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package cmd

import (
	"bytes"
	"io"
	"strings"

	"github.com/lunixbochs/vtclean"
	"github.com/sergi/go-diff/diffmatchpatch"
	"unikraft.com/cli/internal/kvwriter"
	"unikraft.com/cli/internal/prettydiff"
	"unikraft.com/cli/internal/resource"
)

// Diff produces a pretty diff between two sets of resources, writing the
// output to the provided writer.
func Diff(out io.Writer, before []resource.Resource, after []resource.Resource) error {
	before, after = diffOrder(before, after)
	start := &bytes.Buffer{}
	if err := printKV(start, nil, before...); err != nil {
		return err
	}
	end := &bytes.Buffer{}
	if err := printKV(end, nil, after...); err != nil {
		return err
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(
		// clean all ANSI escape sequences from the output before diffing to avoid
		// trying to diff them
		// NOTE: it would be nice to preserve those in the output, but the diffing
		// would have to be done differently
		vtclean.Clean(start.String(), false),
		vtclean.Clean(end.String(), false),
		false,
	)
	tw := kvwriter.KeyValueWriter(out, kvwriter.WithIndent("  "))
	if _, err := io.Copy(tw, strings.NewReader(prettydiff.Render(diffs))); err != nil {
		return err
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	return nil
}

// diffOrder reorders the after resources to match the order of the before
// resources as much as possible. This should hopefully improve the diff
// output.
func diffOrder(before []resource.Resource, after []resource.Resource) ([]resource.Resource, []resource.Resource) {
	afterMap := make(map[string]resource.Resource, len(after))
	for _, r := range after {
		key := r.Key().String()
		afterMap[key] = r
	}
	afterFound := make(map[string]struct{}, len(after))

	var orderedAfter []resource.Resource
	for _, r := range before {
		key := r.Key().String()
		if afterR, ok := afterMap[key]; ok {
			orderedAfter = append(orderedAfter, afterR)
			afterFound[key] = struct{}{}
		}
	}
	for _, r := range after {
		key := r.Key().String()
		if _, ok := afterFound[key]; !ok {
			orderedAfter = append(orderedAfter, r)
		}
	}
	return before, orderedAfter
}
