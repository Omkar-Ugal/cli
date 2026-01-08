// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

//go:build integration

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"
	"mvdan.cc/sh/v3/shell"

	"unikraft.com/x/log"

	"unikraft.com/cli/internal/cmd"
	"unikraft.com/cli/internal/config"
	"unikraft.com/cli/internal/resource"
)

const unikraftCmd = "unikraft"

type testCase struct {
	name     string
	commands []command
	online   bool
}

type command struct {
	args     []string
	allowErr bool
	token    bool
}

var testCases = []testCase{
	{
		name:     "empty",
		commands: []command{{args: []string{unikraftCmd}, allowErr: true}},
	},
	{
		name:     "help",
		commands: []command{{args: []string{unikraftCmd, "--help"}}},
	},
	{
		name:     "version",
		commands: []command{{args: []string{unikraftCmd, "--version"}}},
	},

	{
		name:   "auth",
		online: true,
		commands: []command{
			{args: []string{unikraftCmd, "login"}, token: true},
			{args: []string{unikraftCmd, "profile", "list"}},
			{args: []string{unikraftCmd, "metro", "list"}},
			{args: []string{unikraftCmd, "logout"}},
		},
	},

	{
		name:   "volumes",
		online: true,
		commands: []command{
			{args: []string{unikraftCmd, "login"}, token: true},
			{args: []string{unikraftCmd, "volume", "list"}},
			{args: []string{unikraftCmd, "volume", "create", "--set", "name=test-$UNIQ_VOLUME", "--set", "size=10", "--set", "metro=" + defaultMetro}},
			{args: []string{unikraftCmd, "volume", "list"}},
			{args: []string{unikraftCmd, "volume", "inspect", "test-$UNIQ_VOLUME"}},
			{args: []string{unikraftCmd, "volume", "edit", "test-$UNIQ_VOLUME", "--set", "size=20"}},
			{args: []string{unikraftCmd, "volume", "list"}},
			{args: []string{unikraftCmd, "volume", "inspect", "test-$UNIQ_VOLUME"}},
			{args: []string{unikraftCmd, "volume", "delete", "test-$UNIQ_VOLUME"}},
		},
	},

	{
		name:   "certificates",
		online: true,
		commands: []command{
			{args: []string{unikraftCmd, "login"}, token: true},
			{args: []string{unikraftCmd, "certificate", "list"}},
			{args: []string{unikraftCmd, "certificate", "create", "--set", "name=test-$UNIQ_CERT_A", "--set", "cn=$CERT_A_CN", "--set", "chain=$CERT_A_CHAIN", "--set", "pkey=$CERT_A_KEY", "--set", "metro=" + defaultMetro}},
			{args: []string{unikraftCmd, "certificate", "create", "--set", "name=test-$UNIQ_CERT_B", "--set", "cn=$CERT_B_CN", "--set", "chain=$CERT_B_CHAIN", "--set", "pkey=$CERT_B_KEY", "--set", "metro=" + defaultMetro}},
			{args: []string{unikraftCmd, "certificate", "list"}},
			{args: []string{unikraftCmd, "certificate", "inspect", "test-$UNIQ_CERT_A", "test-$UNIQ_CERT_B"}},
			{args: []string{unikraftCmd, "certificate", "delete", "test-$UNIQ_CERT_A", "test-$UNIQ_CERT_B"}},
		},
	},
}

var (
	token  string
	metros []string
)

const (
	defaultMetro = "test"
)

func init() {
	if v, ok := os.LookupEnv("UKC_TOKEN"); ok {
		token = v
		os.Unsetenv("UKC_TOKEN")
	}

	if v, ok := os.LookupEnv("UKC_METRO"); ok {
		metros = append(metros, v)
		os.Unsetenv("UKC_METRO")
	}
	if v, ok := os.LookupEnv("UKC_METROS"); ok {
		metros = append(metros, strings.Split(v, ",")...)
		os.Unsetenv("UKC_METROS")
	}
}

func TestGolden(t *testing.T) {
	t.Parallel()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.online && token == "" {
				t.Skip("skipping online test that requires UKC_TOKEN")
			}
			if tc.online && len(metros) == 0 {
				t.Skip("skipping online test that requires UKC_METRO/UKC_METROS")
			}

			ctx := t.Context()
			ctx = log.WithLogger(ctx, log.New(t.Output(), log.TextType, log.TraceLevel))

			assert.NotEmpty(t, tc.commands, "no commands specified")

			configdir := filepath.Join(t.TempDir(), "unikraft.d")
			cfg, profile := defaultCfg()
			require.NoError(t, cfg.SaveTo(configdir))

			sandboxPath := filepath.Join(t.TempDir(), "sandbox.json")
			t.Cleanup(func() {
				cfg, err := config.LoadFrom(configdir)
				require.NoError(t, err)
				ctx := config.WithConfig(ctx, cfg)

				sandbox, err := resource.LoadSandbox(sandboxPath, cmd.SandboxedResources...)
				require.NoError(t, err)
				require.NotNil(t, sandbox)

				require.NoError(t, sandbox.Teardown(context.WithoutCancel(ctx)))
			})

			expander := &expander{}
			output := strings.Builder{}
			for i, command := range tc.commands {
				require.NotEmpty(t, command.args, "no command specified")
				var args []string
				if command.args[0] == unikraftCmd {
					args = append(args, "go", "run", ".")
					args = append(args, command.args[1:]...)
				} else {
					assert.Fail(t, "first argument must be %q", unikraftCmd)
					args = command.args
				}
				args = expander.expandArgs(args)

				log.G(ctx).Debug().
					Strs("args", args).
					Msg("executing command")

				cmd := exec.CommandContext(ctx, args[0], args[1:]...)
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				cmd.Env = os.Environ()
				cmd.Env = slices.DeleteFunc(cmd.Env, func(s string) bool {
					return strings.HasPrefix(s, "UNIKRAFT_")
				})
				cmd.Env = append(cmd.Env, "NO_COLOR=1") // color makes golden files harder to read
				cmd.Env = append(cmd.Env, resource.UnikraftSandboxEnv+"="+sandboxPath)
				cmd.Env = append(cmd.Env, config.UnikraftConfigDirEnv+"="+configdir)
				if command.token {
					cmd.Env = append(cmd.Env, "UKC_TOKEN="+token)
				}

				err := cmd.Run()
				var exitErr *exec.ExitError
				var exitCode int
				if errors.As(err, &exitErr) && command.allowErr {
					exitCode = exitErr.ExitCode()
					// ignore exit errors for help commands
					err = nil
				}
				assert.NoError(t, err, "command %q failed", strings.Join(args, " "))

				report := report{
					args:     command.args,
					stdout:   stdout.String(),
					stderr:   stderr.String(),
					exitCode: exitCode,
				}
				report.cleaners = append(report.cleaners, expander.cleaners()...)
				for _, metro := range profile.Metros {
					report.cleaners = append(report.cleaners, cleaner{
						pattern: regexp.MustCompile(regexp.QuoteMeta(metro.Endpoint)),
						repl:    "https://api." + metro.Name + ".unikraft.internal/",
					})
				}
				if i != 0 {
					output.WriteString("\n")
				}
				output.WriteString(report.String())
			}

			golden.Assert(t, output.String(), t.Name())
		})
	}
}

type report struct {
	args     []string
	stdout   string
	stderr   string
	exitCode int
	cleaners []cleaner
}

func (report *report) String() string {
	out := strings.Builder{}

	out.WriteString("$ " + strings.Join(report.args, " ") + "\n\n")
	stdout := report.cleanOutput(report.stdout)
	if len(stdout) > 0 {
		out.WriteString("stdout:\n" + indent(stdout, "\t") + "\n\n")
	}
	stderr := report.cleanOutput(report.stderr)
	if len(stderr) > 0 {
		out.WriteString("stderr:\n" + indent(stderr, "\t") + "\n\n")
	}
	if report.exitCode != 0 {
		out.WriteString("exit code: " + strconv.Itoa(report.exitCode) + "\n\n")
	}

	return strings.TrimSpace(out.String()) + "\n"
}

func (report *report) cleanOutput(s string) string {
	// trim leading and trailing whitespace
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	// apply any necessary cleanup to the output here
	for _, c := range cleaners {
		s = c.pattern.ReplaceAllString(s, c.repl)
	}
	for _, c := range report.cleaners {
		s = c.pattern.ReplaceAllString(s, c.repl)
	}

	return s
}

func indent(s string, indent string) string {
	result := strings.Builder{}
	for line := range strings.Lines(s) {
		if len(strings.TrimSpace(line)) > 0 {
			result.WriteString(indent)
		}
		result.WriteString(line)
	}
	return result.String()
}

type cleaner struct {
	pattern *regexp.Regexp
	repl    string
}

// cleaners are patterns applied to command output to normalize variable data
// so we get consistent golden files.
var cleaners = []cleaner{
	{
		// times like "12:34:56" or "12:34:56PM" change between runs
		pattern: regexp.MustCompile(`\b\d\d?:\d\d:\d\d([AP]M)?\b`),
		repl:    "HH:MM:SS",
	},
	{
		// times like "12:34" or "12:34PM" change between runs
		pattern: regexp.MustCompile(`\b\d\d?:\d\d?([AP]M)?\b`),
		repl:    "HH:MM",
	},
	{
		// dates like "2000-01-02" change between runs
		pattern: regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`),
		repl:    "YYYY-MM-DD",
	},
	{
		// runtime versions like "go1.25.4" change between go releases
		pattern: wordCleaner(runtime.Version()),
		repl:    "goX.Y.Z",
	},
	{
		// platforms like "linux/amd64" change between systems
		pattern: wordCleanerf("%s/%s", runtime.GOOS, runtime.GOARCH),
		repl:    "GOOS/GOARCH",
	},
	{
		// uuids like "12345678-1234-1234-1234-123456789abc" change between runs
		pattern: regexp.MustCompile(`\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`),
		repl:    "12345678-1234-1234-1234-123456789abc",
	},
}

func wordCleaner(word string) *regexp.Regexp {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
}

func wordCleanerf(word string, args ...any) *regexp.Regexp {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(fmt.Sprintf(word, args...)) + `\b`)
}

type expander struct {
	uniq  map[string]string
	certs map[string]*generatedCert
}

type generatedCert struct {
	cn    string
	chain string
	key   string
}

func (e *expander) expandArgs(args []string) []string {
	if e.uniq == nil {
		e.uniq = make(map[string]string)
	}
	if e.certs == nil {
		e.certs = make(map[string]*generatedCert)
	}
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		arg, err := shell.Expand(arg, func(varname string) string {
			prefix, rest, ok := strings.Cut(varname, "_")
			if !ok {
				return ""
			}
			switch prefix {
			case "UNIQ":
				if val, ok := e.uniq[rest]; ok {
					return val
				}
				result := fmt.Sprintf("%x", rand.Text())[:12]
				e.uniq[rest] = result
				return result
			case "CERT":
				name, field, ok := strings.Cut(rest, "_")
				if !ok {
					return ""
				}
				cert, ok := e.certs[name]
				if !ok {
					cert = generateCert()
					e.certs[name] = cert
				}
				switch field {
				case "CN":
					return cert.cn
				case "CHAIN":
					return cert.chain
				case "KEY":
					return cert.key
				}
			}
			return ""
		})
		if err != nil {
			panic(err)
		}
		expanded = append(expanded, arg)
	}
	return expanded
}

func (e *expander) cleaners() []cleaner {
	cleaners := make([]cleaner, 0, len(e.uniq)+len(e.certs))
	for varname, val := range e.uniq {
		cleaners = append(cleaners, cleaner{
			pattern: wordCleaner(val),
			repl:    fmt.Sprintf("<%s>", varname),
		})
	}
	for name, cert := range e.certs {
		// Clean the CN (without trailing dot)
		cn := strings.TrimSuffix(cert.cn, ".")
		cleaners = append(cleaners, cleaner{
			pattern: regexp.MustCompile(regexp.QuoteMeta(cn)),
			repl:    fmt.Sprintf("<CERT_%s_CN>", name),
		})
	}
	return cleaners
}

func defaultCfg() (*config.Config, *config.Profile) {
	profile := &config.Profile{
		Type:  config.ProfileTypeCloud,
		Name:  "default",
		Token: "", // populated via login
	}
	for _, metro := range metros {
		profile.Metros = append(profile.Metros, config.Metro{
			Name:     defaultMetro,
			Endpoint: metro,
			Country:  "xx",
		})
		break
	}
	cfg := &config.Config{
		Profile: "test",
		Profiles: map[string]config.Profile{
			"test": *profile,
		},
	}
	return cfg, profile
}

func generateCert() *generatedCert {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	cn := fmt.Sprintf("test-%x.unikraft.io", rand.Text()[:12])
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		panic(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyDER,
	})

	return &generatedCert{
		cn:    cn + ".",
		chain: string(certPEM),
		key:   string(keyPEM),
	}
}
