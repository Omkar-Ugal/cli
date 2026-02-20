// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSavePreservesCommentsAndDropsUnknownKeys(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.yaml")

	input := strings.TrimSpace(`
# global comment
profile: default
profiles:
  # default profile comment
  default:
    type: cloud
    token: oldtoken
    foobar: remove-me
`) + "\n"

	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config := &Config{
		Path:           path,
		DefaultProfile: "default",
		Profiles: map[string]Profile{
			"default": {
				Type:  ProfileTypeCloud,
				Token: "newtoken",
			},
		},
	}

	if err := config.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(output)

	if !strings.Contains(content, "# global comment") {
		t.Fatalf("expected global comment to be preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "# default profile comment") {
		t.Fatalf("expected profile comment to be preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "token: newtoken") {
		t.Fatalf("expected token to be updated, got:\n%s", content)
	}
	if strings.Contains(content, "foobar") {
		t.Fatalf("expected unknown profile key to be removed, got:\n%s", content)
	}
}
