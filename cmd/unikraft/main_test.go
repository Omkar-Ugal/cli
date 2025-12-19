package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"
	"mvdan.cc/sh/v3/syntax"
)

const unikraftCmd = "unikraft"

type testCase struct {
	name     string
	commands [][]string
	token    bool
}

var testCases = []testCase{
	{
		name:     "empty",
		commands: [][]string{{unikraftCmd}},
	},
	{
		name:     "help",
		commands: [][]string{{unikraftCmd, "--help"}},
	},
	{
		name:     "version",
		commands: [][]string{{unikraftCmd, "--version"}},
	},
	{
		name: "auth",
		commands: [][]string{
			{unikraftCmd, "login"},
			{unikraftCmd, "logout"},
		},
		token: true,
	},
}

func TestGolden(t *testing.T) {
	ctx := t.Context()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, tc.commands, "no commands specified")

			if tc.token && os.Getenv("UKC_TOKEN") == "" {
				t.Skip("skipping test that requires UKC_TOKEN")
			}

			output := strings.Builder{}
			for i, command := range tc.commands {
				require.NotEmpty(t, command, "no command specified")
				var args []string
				if command[0] == unikraftCmd {
					args = append(args, "go", "run", ".")
					args = append(args, command[1:]...)
				} else {
					assert.Fail(t, "first argument must be %q", unikraftCmd)
					args = command
				}

				cmd := exec.CommandContext(ctx, args[0], args[1:]...)
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				cmd.Env = os.Environ()
				cmd.Env = append(cmd.Env, "NO_COLOR=1") // color makes golden files harder to read

				err := cmd.Run()
				var exitErr *exec.ExitError
				var exitCode int
				if errors.As(err, &exitErr) {
					exitCode = exitErr.ExitCode()
					// ignore exit errors for help commands
					err = nil
				}
				assert.NoError(t, err)

				report := report{
					args:     command,
					stdout:   stdout.String(),
					stderr:   stderr.String(),
					exitCode: exitCode,
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
}

func (report *report) String() string {
	out := strings.Builder{}

	args := make([]string, 0, len(report.args))
	for _, arg := range report.args {
		arg, err := syntax.Quote(arg, syntax.LangPOSIX)
		if err != nil {
			panic(err)
		}
		args = append(args, arg)
	}

	out.WriteString("$ " + strings.Join(args, " ") + "\n\n")
	stdout := cleanOutput(report.stdout)
	if len(stdout) > 0 {
		out.WriteString("stdout:\n" + indent(stdout, "\t") + "\n\n")
	}
	stderr := cleanOutput(report.stderr)
	if len(stderr) > 0 {
		out.WriteString("stderr:\n" + indent(stderr, "\t") + "\n\n")
	}
	if report.exitCode != 0 {
		out.WriteString("exit code: " + strconv.Itoa(report.exitCode) + "\n\n")
	}

	return strings.TrimSpace(out.String()) + "\n"
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
		// times like "12:34PM" change between runs
		pattern: regexp.MustCompile(`\b\d\d?:\d\d[AP]M `),
		repl:    "HH:MM ",
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
}

func wordCleaner(word string) *regexp.Regexp {
	return regexp.MustCompile(`\b` + word + `\b`)
}

func wordCleanerf(word string, args ...any) *regexp.Regexp {
	return regexp.MustCompile(`\b` + fmt.Sprintf(word, args...) + `\b`)
}

func cleanOutput(s string) string {
	// trim leading and trailing whitespace
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	// apply any necessary cleanup to the output here
	for _, c := range cleaners {
		s = c.pattern.ReplaceAllString(s, c.repl)
	}

	return s
}
