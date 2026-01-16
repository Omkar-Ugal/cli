module unikraft.com/cli

go 1.25.5

require (
	github.com/MakeNowJust/heredoc v1.0.0
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/alecthomas/kong v1.13.0
	github.com/alecthomas/kong-yaml v0.2.0
	github.com/charmbracelet/bubbletea v1.3.10
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/charmbracelet/x/ansi v0.11.3
	github.com/charmbracelet/x/term v0.2.2
	github.com/containerd/containerd/v2 v2.2.1
	github.com/cpuguy83/go-md2man/v2 v2.0.7
	github.com/distribution/reference v0.6.0
	github.com/docker/go-units v0.5.0
	github.com/ettle/strcase v0.2.0
	github.com/google/uuid v1.6.0
	github.com/juju/ansiterm v1.0.0
	github.com/juju/errors v1.0.0
	github.com/lunixbochs/vtclean v1.0.0
	github.com/mitchellh/copystructure v1.2.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/muesli/termenv v0.16.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/sergi/go-diff v1.4.0
	github.com/stretchr/testify v1.11.1
	github.com/tidwall/gjson v1.18.0
	golang.org/x/net v0.48.0
	golang.org/x/sync v0.19.0
	gopkg.in/yaml.v3 v3.0.1
	gotest.tools/v3 v3.5.2
	mvdan.cc/sh/v3 v3.12.0
	sigs.k8s.io/yaml v1.6.0
	tailscale.com v1.92.5
	unikraft.com/cloud/sdk v0.0.0-20260106155929-09ab9b333e25
	unikraft.com/x/colors v0.0.0-20260105163520-49d071286efd
	unikraft.com/x/kingkong v0.0.0-20260107171923-82c0f5ef829a
	unikraft.com/x/log v0.0.0-20260105163520-49d071286efd
	unikraft.com/x/ptr v0.0.0-20260105163520-49d071286efd
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.3.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/charmbracelet/colorprofile v0.3.3 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.14 // indirect
	github.com/clipperhouse/displaywidth v0.6.1 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dblohm7/wingoes v0.0.0-20240119213807-a09d6be7affa // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/getsentry/sentry-go v0.36.2 // indirect
	github.com/go-json-experiment/json v0.0.0-20250813024750-ebf49471dced // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go4.org/mem v0.0.0-20240501181205-ae6ca9944745 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	unikraft.com/x/guesstermwidth v0.0.0-20251010132444-b0be607b7949 // indirect
)

// Includes a fix for stripping hyperlinks
// https://github.com/lunixbochs/vtclean/pull/15
replace github.com/lunixbochs/vtclean => github.com/jedevc/vtclean v0.0.0-20251216110630-4486acca2b5a
