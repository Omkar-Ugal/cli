// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package yamlmerge

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeYAMLPrunesUnknownKeys(t *testing.T) {
	existing := strings.TrimSpace(`
# global comment
profile: default
profiles:
  # default profile comment
  default:
    type: cloud
    token: oldtoken
    foobar: remove-me
extra:
  enabled: true
`) + "\n"

	desired := strings.TrimSpace(`
profile: default
profiles:
  default:
    type: cloud
    token: newtoken
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
# global comment
profile: default
profiles:
  # default profile comment
  default:
    type: cloud
    token: newtoken
`) + "\n"
	assert.Equal(t, expected, string(output))
}

func TestMergeYAMLMergesSequencesByIndex(t *testing.T) {
	existing := strings.TrimSpace(`
items:
  # first item
  - name: alpha
    value: 1
    extra: drop
  # second item
  - name: beta
    value: 2
`) + "\n"

	desired := strings.TrimSpace(`
items:
  - name: alpha
    value: 10
  - name: beta
    value: 20
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
items:
  # first item
  - name: alpha
    value: 10
  # second item
  - name: beta
    value: 20
`) + "\n"
	assert.Equal(t, expected, string(output))
}

func TestMergeYAMLPreservesAnchors(t *testing.T) {
	existing := strings.TrimSpace(`
profiles: &profiles
  default:
    token: old
`) + "\n"

	desired := strings.TrimSpace(`
profiles:
  default:
    token: new
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
profiles: &profiles
  default:
    token: new
`) + "\n"
	assert.Equal(t, expected, string(output))
}

func TestMergeYAMLAddsAnchorsFromDesired(t *testing.T) {
	existing := strings.TrimSpace(`
profiles:
  default:
    token: old
`) + "\n"

	desired := strings.TrimSpace(`
profiles: &profiles
  default:
    token: new
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
profiles: &profiles
  default:
    token: new
`) + "\n"
	assert.Equal(t, expected, string(output))
}

func TestMergeYAMLPreservesFlowMappingStyle(t *testing.T) {
	existing := strings.TrimSpace(`
settings: {region: eu, retries: 2}
`) + "\n"

	desired := strings.TrimSpace(`
settings:
  region: us
  retries: 3
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
settings: {region: us, retries: 3}
`) + "\n"
	assert.Equal(t, expected, string(output))
}

func TestMergeYAMLPreservesFlowSequenceStyle(t *testing.T) {
	existing := strings.TrimSpace(`
items: [alpha, beta]
`) + "\n"

	desired := strings.TrimSpace(`
items:
  - alpha
  - gamma
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
items: [alpha, gamma]
`) + "\n"
	assert.Equal(t, expected, string(output))
}

func TestMergeYAMLPreservesLiteralStyle(t *testing.T) {
	existing := strings.TrimSpace(`
note: |
  line one
  line two
`) + "\n"

	desired := strings.TrimSpace(`
note: "line one\nline two\nline three"
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
note: |
  line one
  line two
  line three
`) + "\n"
	assert.Equal(t, expected, string(output))
}

func TestMergeYAMLUsesTwoSpaceIndent(t *testing.T) {
	existing := strings.TrimSpace(`
root:
    child: old
`) + "\n"

	desired := strings.TrimSpace(`
root:
  child: new
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
root:
  child: new
`) + "\n"
	assert.Equal(t, expected, string(output))
}

func TestMergeYAMLPreservesTrailingComment(t *testing.T) {
	existing := strings.TrimSpace(`
profile: default
# trailing comment
`) + "\n"

	desired := strings.TrimSpace(`
profile: updated
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
profile: updated
# trailing comment
`) + "\n"
	assert.Equal(t, expected, string(output))
}

func TestMergeYAMLPreservesInlineComment(t *testing.T) {
	existing := strings.TrimSpace(`
profile: default # inline note
`) + "\n"

	desired := strings.TrimSpace(`
profile: updated
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
profile: updated # inline note
`) + "\n"
	assert.Equal(t, expected, string(output))
}

func TestMergeYAMLRemovesDroppedKeyComments(t *testing.T) {
	existing := strings.TrimSpace(`
profiles:
  default:
    token: old
    # removed key comment
    foobar: drop
    type: cloud
`) + "\n"

	desired := strings.TrimSpace(`
profiles:
  default:
    token: new
    type: cloud
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
profiles:
  default:
    token: new
    type: cloud
`) + "\n"
	assert.Equal(t, expected, string(output))
}

func TestMergeYAMLRemovesUnknownProfiles(t *testing.T) {
	existing := strings.TrimSpace(`
profile: dev
profiles:
  # hello world
  dev:
    controlplane: https://controlplane.unikraft.cloud
    metros:
    - country: xx # void
      endpoint: http://api.ukp-stable.apw.unikraft.internal/
      name: ukp-stable
    organization: foo
`) + "\n"

	desired := strings.TrimSpace(`
profile: prod
profiles:
  prod:
    controlplane: https://controlplane.unikraft.cloud
`) + "\n"

	output, err := MergeYAML([]byte(existing), []byte(desired))
	require.NoError(t, err)

	expected := strings.TrimSpace(`
profile: prod
profiles:
  prod:
    controlplane: https://controlplane.unikraft.cloud
`) + "\n"
	assert.Equal(t, expected, string(output))
}
