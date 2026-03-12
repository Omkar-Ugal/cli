# testtui

`testtui` is a small, scriptable driver for the Unikraft Bubble Tea TUI.

It runs the same model as `unikraft tui`, but instead of taking over your
terminal it:

- accepts a stream of simple commands (`key`, `sleep`, `wait`, `snapshot`)
- executes them against the live TUI model
- prints the current rendered view only when you ask for a `snapshot` (text mode)
- can emit an ack event after every command (`--output=json`)

This is useful for debugging TUI behavior, reproducing tricky UI states, and
writing/iterating on golden snapshot tests.

## Run

From the repo root:

```sh
go run ./tools/testtui
```

`testtui` uses the same configuration as the main CLI. If you haven't logged in
yet, resource panels will show the usual `profile not setup` error.

```sh
unikraft login
```

You can optionally start directly on a resource list or detail view:

```sh
# Home
go run ./tools/testtui

# Resource list (either singular or plural works)
go run ./tools/testtui instances

# Resource detail
go run ./tools/testtui instances fra/my-instance
```

By default it reads commands from stdin (`--script -`).

Useful flags:

- `--config` (or `UNIKRAFT_CONFIG`): path to the config file
- `--profile` (or `UNIKRAFT_PROFILE`): select the active profile
- `--output`: `text` (default) or `json`
- `--color` / `--no-color`: force-enable or force-disable color output
- `--width`, `--height`: snapshot dimensions (lines are truncated to fit)
- `--wait-timeout`: default timeout for `wait ...`
- `--wait-interval`: polling interval for `wait ...` (set to `0` for update-driven checks only)
- `--script <path>`: read commands from a file (use `-` for stdin)
- `--cmd <line>`: run a command line before the script (repeatable)

## Language

Each line is one command. Whitespace is flexible.

- Blank lines are ignored.
- `#` starts a comment (either as a whole line, or after a command).

### Commands

```
snapshot
sleep <time.Duration>
key <key>
wait <expr>
```

`time.Duration` is parsed by Go (e.g. `150ms`, `2s`, `1m`).

In `--output=text` (default):

- `snapshot` prints the current view to stdout (ANSI is stripped).
- other commands are silent.

In `--output=json`:

- every non-empty, non-comment line emits one JSON object on stdout
- `snapshot` includes the view text in the JSON event (and errors include the last view)

## JSON output

With `--output=json`, stdout is newline-delimited JSON (JSONL): one object per
executed line.

Event shape:

```json
{
  "seq": 1,
  "line": 12,
  "cmd": "wait contains(\"Home\")",
  "kind": "wait",
  "elapsed_ms": 123,
  "snapshot": "... (only for snapshot) ...",
  "error": "... (only on errors) ..."
}
```

Notes:

- `seq` increments per executed line (comments/blank lines do not emit events).
- `line` is only set when reading from a script/stream (stdin or `--script`).
- `snapshot` is present for the `snapshot` command.
- On failures, the event includes `error` and `snapshot` (the last rendered view).

### Wait expressions

Atoms:

- `contains("string")`

Operators:

- `not <expr>`
- `<expr> and <expr>`
- `<expr> or <expr>`
- parentheses: `(<expr>)`

Precedence is `not` > `and` > `or`.

Strings use double quotes and Go-style escapes (`\"`, `\n`, `\\`, ...).

Examples:

```
wait contains("Home > instances")
wait not contains("Loading...")
wait contains("alpha") and contains("bravo")
wait (not contains("ERROR")) or contains("No results")
```

### Keys

`key <key>` sends a Bubble Tea key press.

Supported named keys:

`enter`, `tab`, `backspace`, `esc`, `space`, `up`, `down`, `left`, `right`,
`pgup`, `pgdown`, `home`, `end`, `insert`, `delete`.

Modifiers are written with `+`:

`ctrl+<key>`, `alt+<key>`, `shift+<key>`, `meta+<key>`, `super+<key>`, `hyper+<key>`.

Examples:

```
key r
key enter
key tab
key shift+tab
key ctrl+c
key ?
```

## Examples

Show the home screen:

```sh
cat <<'EOF' | go run ./tools/testtui
snapshot
EOF
```

Same, but JSON output (extract only snapshots):

```sh
cat <<'EOF' | go run ./tools/testtui --output=json | jq -r 'select(.snapshot != null) | .snapshot'
snapshot
EOF
```

Start on the instances list and snapshot once it's settled:

```sh
cat <<'EOF' | go run ./tools/testtui instances
wait not contains("Loading...")
snapshot
EOF
```

Toggle help:

```sh
cat <<'EOF' | go run ./tools/testtui
snapshot
key ?
snapshot
EOF
```

## Long-lived sessions (named pipe)

For agent-style workflows it's useful to keep `testtui` running and send it
commands over time.

You can do that with a FIFO (named pipe), but you must keep a writer open.
If the last writer closes, `testtui` will read EOF and exit.

Two-terminal setup:

Terminal A:

```sh
mkfifo /tmp/testtui.pipe
go run ./tools/testtui --output=json --script /tmp/testtui.pipe --width 80 --height 24 \
  | tee /tmp/testtui.jsonl
```

To see only snapshots in a human-readable form:

```sh
tail -f /tmp/testtui.jsonl | jq -r 'select(.snapshot != null) | .snapshot'
```

Terminal B (keep a writer open on FD 3):

```sh
exec 3>/tmp/testtui.pipe

printf 'snapshot\n' >&3
printf 'key ?\nsnapshot\n' >&3
```

If you need to send one-off commands from many short-lived processes (common for
agents), keep the pipe open with a dedicated "keeper" process:

```sh
# Keep the FIFO open (no output is written).
tail -f /dev/null > /tmp/testtui.pipe &

# Now you can send commands without keeping an FD open.
printf 'snapshot\n' > /tmp/testtui.pipe
printf 'key ?\nsnapshot\n' > /tmp/testtui.pipe
```

To stop the tool you can either kill it, or send `key ctrl+c` (the TUI quit
binding).
