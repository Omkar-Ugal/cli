# Unikraft CLI

> [!WARNING]
>
> Under construction! :construction_worker:

This repository contains the code for a work-in-progress, internal tool for
managing the [Unikraft Cloud Platform](https://unikraft.com/). Eventually, this will replace [kraft](https://github.com/unikraft/kraftkit).

> [!TIP]
>
> Internal preview docs are available at <https://cli-preview.ukp-stable.apw.unikraft.internal/docs/cli-new/unikraft>.
> See <https://github.com/unikraft-cloud/www/pull/497> for more info.

## Setup

To clone the repository, run:

```bash
git clone https://github.com/unikraft/cli.git
cd cli
```

This repo uses `task` (from <https://taskfile.dev/>) to manage builds and other tasks.
To install `task`, follow the [installation guide](https://taskfile.dev/docs/installation).

You'll also need to enable `task`'s [Remote Taskfiles](https://taskfile.dev/docs/experiments/remote-taskfiles) feature:

```bash
export TASK_X_REMOTE_TASKFILES=1
```

## Building

To build the `unikraft` CLI, ensure you have Go installed, then run:

```bash
task cli
```

This will build the `unikraft` binary, and place it in the `dist/` directory.
You should add this directory to your `PATH` environment variable to use it.

For example:

```bash
export PATH=$PATH:$PWD/dist
```

## Usage

To use the `unikraft` CLI, you'll first need to login:

```bash
unikraft login
```

Alternatively, you can manually create a profile:

```yaml
# Linux: ~/.config/unikraft/config.yaml
# MacOS: ~/Library/Application\ Support/unikraft/config.yaml
profile: default
profiles:
  default:
    type: cloud
    name: default
    organization: <org-name>
    token: <api-token>
    metros:
      - name: <metro-name> # e.g. fra
        endpoint: <metro-endpoint> # e.g. https://api.fra.unikraft.cloud
        country: <metro-country> # e.g. de
```

Then you'll be able to run commands against the platform. For example, to list
all available metros:

```bash
unikraft metro list
```

To see all available commands, run:

```bash
unikraft --help
```

### Auto-completion

To see instructions to enable auto-completion for your shell, run:

```bash
unikraft completion
```

## Documentation

To build the documentation, run:

```bash
task docs
```

This builds:

- Markdown documentation in `dist/docs/`
- Man pages in `dist/man/`

## Development

Building the CLI for development is exactly the same as above.

We use a variety of tools to ensure code quality:

- `task lint` - runs golangci-lint
- `task test` - runs unit tests
- `task integration` - runs integration tests
  - `task integration-update` - updates the integration test golden files

These are also run as part of the CI process, however, you can run them locally
to ensure your changes pass before pushing.
