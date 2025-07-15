# SPDX-License-Identifier: BSD-3-Clause
# Copyright (c) 2025, Unikraft GmbH and The Unikraft CLI Authors.
# Licensed under the BSD-3-Clause License (the "License").
# You may not use this file except in compliance with the License.

# Prelude
SHELL              := bash
.DELETE_ON_ERROR:
.SHELLFLAGS        := -eu -o pipefail -c
.DEFAULT_GOAL      := all
Q                  ?= @

# Directories
WORKDIR            ?= $(CURDIR)
DISTDIR            ?= $(WORKDIR)/dist
GOMOD              ?= unikraft.com/cli

# Add a special version tag for pull requests
ifneq ($(shell grep 'refs/pull' $(WORKDIR)/.git/FETCH_HEAD),)
HASH_COMMIT        ?= HEAD
HASH               += pr-$(shell cat $(WORKDIR)/.git/FETCH_HEAD | awk -F/ '{print $$3}')
endif

# Calculate the project version based on git history
ifeq ($(HASH),)
HASH_COMMIT        ?= HEAD
HASH               ?= $(shell git update-index -q --refresh && \
                              git describe --tags)
# Others can't be dirty by definition
ifneq ($(HASH_COMMIT),HEAD)
HASH_COMMIT        ?= HEAD
endif
DIRTY              ?= $(shell git update-index -q --refresh && \
                              git diff-index --quiet HEAD -- $(WORKDIR) || \
                              echo "-dirty")
endif
VERSION            ?= $(HASH)$(DIRTY)
GIT_SHA            ?= $(shell git update-index -q --refresh && \
                              git rev-parse --short HEAD)

# Arguments
BIN                ?= unikraft

# Tools
GO                 ?= go
CAT                ?= cat

# Go tools
GOCILINT_VERSION   ?= v2.2.2
GOCILINT           ?= $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOCILINT_VERSION)
GORELEASER_VERSION ?= v2.11.0
GORELEASER         ?= $(GO) run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
YTT_VERSION        ?= v0.52.0
YTT                ?= $(GO) run carvel.dev/ytt/cmd/ytt@$(YTT_VERSION)

# Targets
.PHONY: all
all: tidy $(BIN)
all: ## All targets.

.PHONY: build
build: $(WORKDIR)/.goreleaser.yaml
build: ## Build a release.
	$(Q)$(GORELEASER) release --config $(WORKDIR)/.goreleaser.yaml --skip publish --skip validate --clean

.PHONY: snapshot
build: $(WORKDIR)/.goreleaser.yaml
snapshot: ## Build a snapshot release.
	$(Q)$(GORELEASER) release --config $(WORKDIR)/.goreleaser.yaml --snapshot --skip validate --clean

.PHONY: $(WORKDIR)/.goreleaser.yaml
$(WORKDIR)/.goreleaser.yaml: $(WORKDIR)/.goreleaser.ytt
	$(Q)$(CAT) $< | $(YTT) --data-values-env YTT -f- > $@

ifeq ($(DEBUG),y)
$(BIN): GO_GCFLAGS ?= -N -l
else
$(BIN): GO_LDFLAGS ?= -s -w -extldflags -static-pie
endif
$(BIN): GO_LDFLAGS += -X "$(GOMOD)/internal/version.Version=$(VERSION)"
$(BIN): GO_LDFLAGS += -X "$(GOMOD)/internal/version.Commit=$(GIT_SHA)"
$(BIN): GO_LDFLAGS += -X "$(GOMOD)/internal/version.BuildTime=$(shell date)"
$(BIN): tidy
	$(Q)\
		GOOS=$(GOOS) \
		GOARCH=$(GOARCH) \
	$(GO) build \
		-v \
		-buildmode=pie \
		-gcflags=all='$(GO_GCFLAGS)' \
		-ldflags='$(GO_LDFLAGS)' \
		-o $(DISTDIR)/$(@) \
		$(WORKDIR)/cmd/$(@)

.PHONY: fmt
fmt: ## Format all files according to linting preferences.
	$(Q)$(GOCILINT) fmt

.PHONY: cicheck
cicheck: ## Run CI checks.
	$(GOCILINT) run

.PHONY: tidy
tidy: ## Tidy Go modules.
	$(Q)$(GO) mod tidy

.PHONY: show-version
show-version: ## Show the current version of the CLI.
	$(Q)echo "$(VERSION)"

.PHONY: help
help: ## Show this help menu and exit.
	$(Q)awk 'BEGIN { \
		FS = ":.*##"; \
		printf "Unikraft CLI: Developer Build Targets.\n\n"; \
		printf "\033[1mUSAGE\033[0m\n"; \
		printf "  make [VAR=... [VAR=...]] \033[36mTARGET\033[0m\n\n"; \
		printf "\033[1mTARGETS\033[0m\n"; \
	} \
	/^[a-zA-Z0-9_-]+:.*?##/ { \
		printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2 \
	} \
	/^##@/ { \
		printf "\n\033[1m%s\033[0m\n", substr($$0, 5) \
	} ' $(MAKEFILE_LIST)
