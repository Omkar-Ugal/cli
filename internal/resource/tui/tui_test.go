// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package tui_test

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"

	"unikraft.com/cli/internal/cmd"
	"unikraft.com/cli/internal/resource"
	resourcet "unikraft.com/cli/internal/resource/testing"
	tui "unikraft.com/cli/internal/resource/tui"
	"unikraft.com/cli/internal/tui/testkit"
	"unikraft.com/cli/internal/tui/uitui"
)

func TestMain(m *testing.M) {
	lipgloss.Writer.Profile = colorprofile.Ascii
	os.Exit(m.Run())
}

type subpanelTestResource struct {
	resourcet.TestResource
}

func (subpanelTestResource) Subpanels(ctx context.Context, key string) []tea.Model {
	readerFunc := func(ctx context.Context) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("sample logs\n")), nil
	}
	return []tea.Model{tui.NewReaderPanelFunc(ctx, "Logs", readerFunc, true)}
}

func setupTestEnv() *resourcet.TestEnv {
	env := resourcet.NewTestEnv()
	for _, r := range resourcet.BaseTestStore() {
		env.Add(r)
	}
	return env
}

func TestTUIBasic(t *testing.T) {
	env := setupTestEnv()
	ctx := resourcet.WithTestEnv(context.Background(), env)

	registry := resource.NewRegistry()
	desc := registry.Register(resourcet.TestResource{}, resourcet.TestResource{})
	panel := tui.NewListPanel(ctx, registry, desc)
	model := uitui.NewModel(panel)

	runner := testkit.New(model, 80, 18)
	defer runner.Stop()

	listView, err := runner.WaitUntilSnapshot(2*time.Second, 0, func(s string) bool {
		return strings.Contains(s, "id-test1") && strings.Contains(s, "id-test2") && !strings.Contains(s, "Loading...")
	})
	require.NoErrorf(t, err, "wait failed: %v\n\nlast snapshot:\n%s", err, runner.Snapshot())

	runner.PressKeys(tea.Key{Code: tea.KeyEnter})
	detailView, err := runner.WaitUntilSnapshot(2*time.Second, 0, func(s string) bool {
		return strings.Contains(s, "tests > test test1") && strings.Contains(s, "settings") && strings.Contains(s, "authors")
	})
	require.NoErrorf(t, err, "wait failed: %v\n\nlast snapshot:\n%s", err, runner.Snapshot())

	output := strings.Join([]string{
		"=== list ===",
		listView,
		"",
		"=== detail ===",
		detailView,
	}, "\n")

	golden.Assert(t, output, t.Name(), "\n"+output)
}

func TestTUIWidePanels(t *testing.T) {
	env := setupTestEnv()
	ctx := resourcet.WithTestEnv(context.Background(), env)

	registry := resource.NewRegistry()
	desc := registry.Register(resourcet.TestResource{}, resourcet.TestResource{})
	panel := tui.NewListPanel(ctx, registry, desc)
	model := uitui.NewModel(panel)

	runner := testkit.New(model, 140, 18)
	defer runner.Stop()

	listView, err := runner.WaitUntilSnapshot(2*time.Second, 0, func(s string) bool {
		return strings.Contains(s, "id-test1") && strings.Contains(s, "id-test2") && !strings.Contains(s, "Loading...")
	})
	require.NoErrorf(t, err, "wait failed: %v\n\nlast snapshot:\n%s", err, runner.Snapshot())

	runner.PressKeys(tea.Key{Code: tea.KeyEnter})
	detailView, err := runner.WaitUntilSnapshot(2*time.Second, 0, func(s string) bool {
		return strings.Contains(s, "tests > test test1") && strings.Contains(s, "settings") && strings.Contains(s, "authors")
	})
	require.NoErrorf(t, err, "wait failed: %v\n\nlast snapshot:\n%s", err, runner.Snapshot())

	output := strings.Join([]string{
		"=== list ===",
		listView,
		"",
		"=== list+detail ===",
		detailView,
	}, "\n")

	golden.Assert(t, output, t.Name(), "\n"+output)
}

func TestTUIHomePanel(t *testing.T) {
	ctx := context.Background()

	model, err := cmd.NewTUIModel(ctx, "", "")
	require.NoError(t, err)

	runner := testkit.New(model, 80, 22)
	defer runner.Stop()

	view, err := runner.WaitUntilSnapshot(3*time.Second, 0, func(s string) bool {
		return strings.Contains(s, "c'_'o  .--'")
	})
	require.NoErrorf(t, err, "wait failed: %v\n\nlast snapshot:\n%s", err, runner.Snapshot())

	output := strings.Join([]string{
		"=== home ===",
		view,
	}, "\n")
	output += "\n"

	golden.Assert(t, output, t.Name(), "\n"+output)
}

func TestTUIDetailWithSubpanel(t *testing.T) {
	env := setupTestEnv()
	ctx := resourcet.WithTestEnv(context.Background(), env)

	registry := resource.NewRegistry()
	desc := registry.Register(subpanelTestResource{}, subpanelTestResource{})
	listPanel := tui.NewListPanel(ctx, registry, desc)
	model := uitui.NewModel(listPanel)
	runner := testkit.New(model, 80, 30)
	defer runner.Stop()

	listView, err := runner.WaitUntilSnapshot(2*time.Second, 0, func(s string) bool {
		return strings.Contains(s, "id-test1") && strings.Contains(s, "id-test2") && !strings.Contains(s, "Loading...")
	})
	require.NoErrorf(t, err, "wait failed: %v\n\nlast snapshot:\n%s", err, runner.Snapshot())

	runner.PressKeys(tea.Key{Code: tea.KeyEnter})
	detailView, err := runner.WaitUntilSnapshot(2*time.Second, 0, func(s string) bool {
		return strings.Contains(s, "tests > test test1") &&
			strings.Contains(s, "sample logs") &&
			strings.Contains(s, "Field") &&
			strings.Contains(s, "id-test1") &&
			!strings.Contains(s, "Loading...")
	})
	require.NoErrorf(t, err, "wait failed: %v\n\nlast snapshot:\n%s", err, runner.Snapshot())

	runner.PressKeys(tea.Key{Code: tea.KeyTab})
	logsView, err := runner.WaitUntilSnapshot(2*time.Second, 0, func(s string) bool {
		return strings.Contains(s, "sample logs") &&
			strings.Contains(s, "Field") &&
			strings.Contains(s, "id-test1") &&
			!strings.Contains(s, "Loading...")
	})
	require.NoErrorf(t, err, "wait failed: %v\n\nlast snapshot:\n%s", err, runner.Snapshot())

	output := strings.Join([]string{
		"=== list ===",
		listView,
		"",
		"=== detail (main focused) ===",
		detailView,
		"",
		"=== detail (logs focused) ===",
		logsView,
	}, "\n")
	output += "\n"

	golden.Assert(t, output, t.Name(), "\n"+output)
}
