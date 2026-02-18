// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package yamlmerge

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

// MergeYAML merges desired into existing while preserving comments.
// All keys not present in desired are removed. Sequences are merged by index
// when lengths match.
func MergeYAML(existing, desired []byte) ([]byte, error) {
	if len(desired) == 0 {
		return existing, nil
	}

	desiredDoc, err := decodeDocument(desired)
	if err != nil {
		return nil, err
	}
	if len(existing) == 0 {
		return encodeDocument(desiredDoc)
	}
	desiredRoot := rootNode(desiredDoc)

	existingDoc, err := decodeDocument(existing)
	if err != nil {
		return nil, err
	}
	existingRoot := rootNode(existingDoc)

	if existingRoot == nil {
		return encodeDocument(desiredDoc)
	}
	if desiredRoot == nil {
		return encodeDocument(existingDoc)
	}

	mergeNode(existingRoot, desiredRoot)
	return encodeDocument(existingDoc)
}

func mergeNode(existing *yaml.Node, desired *yaml.Node) {
	if existing.Kind == yaml.MappingNode && desired.Kind == yaml.MappingNode {
		mergeMap(existing, desired)
		return
	}
	if existing.Kind == yaml.SequenceNode && desired.Kind == yaml.SequenceNode {
		mergeSeq(existing, desired)
		return
	}
	applyData(existing, desired)
}

func mergeMap(existing *yaml.Node, desired *yaml.Node) {
	if existing.Kind != yaml.MappingNode || desired.Kind != yaml.MappingNode {
		applyData(existing, desired)
		return
	}

	applyMetadata(existing, desired)

	index := make(map[string]int, len(existing.Content)/2)
	for i := 0; i < len(existing.Content); i += 2 {
		key := existing.Content[i]
		index[key.Value] = i
	}

	desiredKeys := make(map[string]struct{}, len(desired.Content)/2)
	for i := 0; i < len(desired.Content); i += 2 {
		key := desired.Content[i]
		value := desired.Content[i+1]
		desiredKeys[key.Value] = struct{}{}

		if pos, ok := index[key.Value]; ok {
			mergeNode(existing.Content[pos+1], value)
			continue
		}
		existing.Content = append(existing.Content, key, value)
	}

	filtered := make([]*yaml.Node, 0, len(existing.Content))
	for i := 0; i < len(existing.Content); i += 2 {
		key := existing.Content[i]
		if _, ok := desiredKeys[key.Value]; ok {
			filtered = append(filtered, existing.Content[i], existing.Content[i+1])
		}
	}
	existing.Content = filtered
}

func mergeSeq(existing *yaml.Node, desired *yaml.Node) {
	if len(existing.Content) != len(desired.Content) {
		applyData(existing, desired)
		return
	}

	// Merge sequence items by index to keep per-item comments stable.
	applyMetadata(existing, desired)

	for i := range desired.Content {
		mergeNode(existing.Content[i], desired.Content[i])
	}
}

func applyData(existing *yaml.Node, desired *yaml.Node) {
	// Preserve metadata while swapping out the node value.
	headComment := existing.HeadComment
	lineComment := existing.LineComment
	footComment := existing.FootComment
	prevStyle := existing.Style
	prevAnchor := existing.Anchor
	prevTag := existing.Tag
	prevKind := existing.Kind
	prevValue := existing.Value
	*existing = *desired
	if prevStyle != 0 && prevKind == desired.Kind {
		existing.Style = prevStyle
	}
	if prevKind == yaml.ScalarNode && (prevStyle == yaml.LiteralStyle || prevStyle == yaml.FoldedStyle) {
		if strings.HasSuffix(prevValue, "\n") && !strings.HasSuffix(existing.Value, "\n") {
			existing.Value += "\n"
		}
	}
	if prevAnchor != "" {
		existing.Anchor = prevAnchor
	}
	if desired.Tag == "" {
		existing.Tag = prevTag
	}
	existing.HeadComment = headComment
	existing.LineComment = lineComment
	existing.FootComment = footComment
}

func applyMetadata(existing *yaml.Node, desired *yaml.Node) {
	if desired.Tag != "" {
		existing.Tag = desired.Tag
	}
	if existing.Style == 0 && desired.Style != 0 {
		existing.Style = desired.Style
	}
	if existing.Anchor == "" && desired.Anchor != "" {
		existing.Anchor = desired.Anchor
	}
}

func rootNode(doc *yaml.Node) *yaml.Node {
	if doc == nil {
		return nil
	}
	if doc.Kind == yaml.DocumentNode {
		if len(doc.Content) == 0 {
			return nil
		}
		return doc.Content[0]
	}
	return doc
}

func decodeDocument(input []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(input, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func encodeDocument(doc *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		if err := encoder.Encode(doc.Content[0]); err != nil {
			encoder.Close()
			return nil, err
		}
	} else {
		if err := encoder.Encode(doc); err != nil {
			encoder.Close()
			return nil, err
		}
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
