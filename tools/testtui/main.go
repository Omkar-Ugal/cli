// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2026, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/kong"
	"github.com/charmbracelet/colorprofile"
	"unikraft.com/x/kingkong"

	"unikraft.com/cli/internal/cmd"
	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/tui/testkit"
)

type CLI struct {
	Config  string `name:"config" env:"UNIKRAFT_CONFIG" help:"Path to the configuration file." placeholder:"file"`
	Profile string `name:"profile" env:"UNIKRAFT_PROFILE" help:"Set the current profile." placeholder:"name"`

	Resource string `arg:"" optional:"" help:"Resource type to browse."`
	Name     string `arg:"" optional:"" help:"Resource key to open."`

	Width        int           `name:"width" help:"Window width." default:"80"`
	Height       int           `name:"height" help:"Window height." default:"24"`
	WaitTimeout  time.Duration `name:"wait-timeout" help:"Default timeout for wait commands." default:"5s"`
	WaitInterval time.Duration `name:"wait-interval" help:"Polling interval for wait commands." default:"25ms"`

	Color *bool `name:"color" negatable:"" help:"Force color output (or disable with --no-color)."`

	Script *os.File `name:"script" help:"Read commands from file ('-' for stdin)." default:"-" placeholder:"file"`
	Cmd    []string `name:"cmd" help:"Command line to run (repeatable)."`

	Output string `name:"output" help:"Output format." enum:"text,json" default:"text"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var cli CLI
	kctx := kong.Parse(&cli,
		kong.Name("testtui"),
		kong.Help(kingkong.HelpPrinter("")),
		kong.Description("Preview `unikraft tui` rendering via scripted Bubble Tea inputs."),
		kong.UsageOnError(),
	)

	if err := run(ctx, &cli); err != nil {
		kctx.FatalIfErrorf(err)
	}
}

func run(ctx context.Context, cli *CLI) error {
	if cli.Color != nil {
		if *cli.Color {
			lipgloss.Writer.Profile = colorprofile.TrueColor
		} else {
			lipgloss.Writer.Profile = colorprofile.Ascii
		}
	}

	configPath := cli.Config
	if configPath == "" {
		p, err := config.ConfigFilePath()
		if err != nil {
			return err
		}
		configPath = p
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &config.Config{Path: configPath}
	}
	if cli.Profile != "" {
		cfg.OverrideCurrentProfile(cli.Profile)
	}
	ctx = config.WithConfig(ctx, cfg)

	model, err := cmd.NewTUIModel(ctx, cli.Resource, cli.Name)
	if err != nil {
		return err
	}

	runner := testkit.New(model, cli.Width, cli.Height)
	defer runner.Stop()

	out := newOutput(cli.Output)
	exec := &executor{
		runner:       runner,
		out:          out,
		waitTimeout:  cli.WaitTimeout,
		waitInterval: cli.WaitInterval,
	}

	for _, line := range cli.Cmd {
		if err := exec.runLine(ctx, 0, line); err != nil {
			return err
		}
	}

	r := cli.Script
	if r == nil {
		r = os.Stdin
	}
	if cli.Script != nil && cli.Script != os.Stdin {
		defer cli.Script.Close()
	}
	return runStream(ctx, exec, r)
}

type outputMode int

const (
	outputText outputMode = iota
	outputJSON
)

type output struct {
	mode outputMode
	enc  *json.Encoder
	seq  int
}

func newOutput(mode string) *output {
	out := &output{mode: outputText}
	if strings.EqualFold(mode, "json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		out.mode = outputJSON
		out.enc = enc
	}
	return out
}

type event struct {
	Seq int `json:"seq"`

	// Line is the 1-based line number in the script/stream.
	Line int `json:"line,omitempty"`

	Cmd  string `json:"cmd"`
	Kind string `json:"kind"`

	ElapsedMS int64  `json:"elapsed_ms"`
	Error     string `json:"error,omitempty"`

	// Snapshot is included for the `snapshot` command and on errors.
	Snapshot string `json:"snapshot,omitempty"`
}

type executor struct {
	runner       *testkit.Runner
	out          *output
	waitTimeout  time.Duration
	waitInterval time.Duration
}

func runStream(ctx context.Context, exec *executor, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		if err := exec.runLine(ctx, lineNo, scanner.Text()); err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (e *executor) runLine(ctx context.Context, lineNo int, raw string) error {
	_ = ctx

	cmd, skip, err := parseLine(raw)
	if skip && err == nil {
		return nil
	}

	start := time.Now()
	cmdText := strings.TrimSpace(raw)

	kind := "parse"
	var snapshot string
	var execErr error
	if err != nil {
		execErr = err
	} else {
		kind = cmd.Kind.String()
		snapshot, execErr = e.runCommand(cmd)
	}
	elapsed := time.Since(start)

	if e.out != nil && e.out.mode == outputJSON {
		e.out.seq++
		ev := event{
			Seq:       e.out.seq,
			Cmd:       cmdText,
			Kind:      kind,
			ElapsedMS: elapsed.Milliseconds(),
		}
		if lineNo > 0 {
			ev.Line = lineNo
		}
		if snapshot != "" {
			ev.Snapshot = snapshot
		}
		if execErr != nil {
			ev.Error = execErr.Error()
			if e.runner != nil {
				ev.Snapshot = e.runner.Snapshot()
			}
		}
		if err := e.out.enc.Encode(ev); err != nil {
			return err
		}
	}

	return execErr
}

func (e *executor) runCommand(c Command) (string, error) {
	switch c.Kind {
	case CommandSleep:
		time.Sleep(c.Sleep)
		return "", nil
	case CommandKey:
		e.runner.PressKeys(c.Key)
		return "", nil
	case CommandWait:
		if c.Wait == nil {
			return "", fmt.Errorf("wait expression is nil")
		}
		if err := e.runner.WaitUntil(e.waitTimeout, e.waitInterval, c.Wait.Eval); err != nil {
			return "", err
		}
		return "", nil
	case CommandSnapshot:
		snap := e.runner.Snapshot()
		if e.out == nil || e.out.mode == outputText {
			fmt.Fprintln(os.Stdout, snap)
			return "", nil
		}
		return snap, nil
	default:
		return "", fmt.Errorf("unsupported command")
	}
}
