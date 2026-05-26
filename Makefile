# path: Makefile

# DQ Platform — top-level developer commands.
#
# Targets follow the catalog proposed in
# studies/foundation/02-monorepo-topology.md §"Local Development".
# Substrate-touching targets (up, down, test, lint, lint-engine,
# test-engine) are real in Phase 2. Targets that depend on tools
# not yet scaffolded (lint-rules, dry-run-rules) print an explicit
# deferral message naming the phase that produces the missing
# tool.

SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

# Compose project name keeps local volumes / networks namespaced.
COMPOSE_PROJECT_NAME ?= dq-platform
COMPOSE := docker compose -p $(COMPOSE_PROJECT_NAME)

.PHONY: help \
	lint test \
	lint-engine test-engine test-engine-integration \
	lint-tools test-tools test-tools-manifest-integration \
	lint-rules dry-run-rules \
	sync-schema \
	build-lint build-engine build-manifest build-engine-image \
	build-dryrun \
	check-tag-scope \
	up down \
	smoke-substrate \
	validate-deploy

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z][a-zA-Z0-9_-]+:.*?## / { printf "  \033[1m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

lint: lint-engine lint-tools ## Lint engine + tools modules (go vet across each).

test: smoke-substrate test-engine test-tools ## Run substrate smoke tests + go tests across modules.

lint-engine: ## go vet across the engine module.
	@cd engine && go vet ./...

test-engine: ## go test across the engine module (unit tests only).
	@cd engine && go test ./...

test-engine-integration: ## go test -tags integration across the engine module; requires `make up` first.
	@cd engine && go test -tags integration ./...

lint-tools: ## go vet across every module under tools/.
	@cd tools/lint && go vet ./...
	@cd tools/manifest && go vet ./...

test-tools: ## go test across every module under tools/.
	@cd tools/lint && go test ./...
	@cd tools/manifest && go test ./...

test-tools-manifest-integration: ## go test -tags integration on tools/manifest; requires `make up` first.
	@cd tools/manifest && go test -tags integration ./...

build-lint: ## Build the dq-lint binary at bin/dq-lint.
	@mkdir -p bin
	@cd tools/lint && go build -o ../../bin/dq-lint .

build-engine: ## Build the dq-engine binary at bin/dq-engine.
	@mkdir -p bin
	@cd engine && go build -o ../bin/dq-engine ./cmd/dq-engine

build-manifest: ## Build the dq-manifest binary at bin/dq-manifest.
	@mkdir -p bin
	@cd tools/manifest && go build -o ../../bin/dq-manifest .

build-dryrun: ## Build the dq-dryrun binary at bin/dq-dryrun.
	@mkdir -p bin
	@cd tools/dryrun && go build -o ../../bin/dq-dryrun .

# Image tag derivation per ADR-0042 Clause 3: stripping the
# `engine-v` prefix from a git tag yields the image tag (e.g.,
# `engine-v1.2.0` → `dq-engine:1.2.0`). When no `engine-v*` tag
# points at HEAD, fall back to the short SHA — useful for PR
# builds where no release tag exists yet.
ENGINE_GIT_TAG := $(shell git describe --tags --match 'engine-v*' --exact-match 2>/dev/null)
ENGINE_IMAGE_TAG ?= $(if $(ENGINE_GIT_TAG),$(patsubst engine-v%,%,$(ENGINE_GIT_TAG)),$(shell git rev-parse --short HEAD))
ENGINE_IMAGE_NAME ?= dq-engine

build-engine-image: ## Build the dq-engine container image per ADR-0042 Clause 1; tag from git per ADR-0042 Clause 3.
	@docker build -t $(ENGINE_IMAGE_NAME):$(ENGINE_IMAGE_TAG) ./engine

check-tag-scope: ## Validate that TAG=<workspace-v…> diffs cleanly against its prior matching tag (ADR-0042 Clause 3 / B2-29 gate).
	@if [ -z "$(TAG)" ]; then echo "usage: make check-tag-scope TAG=<workspace-v…>" >&2; exit 1; fi
	@bash scripts/check-tag-scope.sh "$(TAG)"

lint-rules: build-lint ## Validate every rule YAML against the schema mirror (per ADR-0001).
	@./bin/dq-lint -schema rules/_schema/v1.schema.json -rules rules

dry-run-rules: build-dryrun ## Dry-run every set-mode BigQuery rule via dq-dryrun (per ADR-0029 + B2-11). Requires `make up` to seed the local emulator.
	@./bin/dq-dryrun -rules rules -bigquery-project dq-local -bigquery-emulator-host $${BIGQUERY_EMULATOR_HOST:-localhost:9050}

sync-schema: ## Mechanically derive the rules schema + catalog mirrors from the engine source (ADR-0001 C3, ADR-0022 §C-B0S2.1).
	@for src in engine/internal/dsl/schema/v*.schema.json; do \
		base="$$(basename $$src)"; \
		dst="rules/_schema/$$base"; \
		cp -p "$$src" "$$dst" && echo "synced $$src -> $$dst"; \
	done
	@for src in engine/internal/dsl/catalog/v*.yaml; do \
		[ -e "$$src" ] || continue; \
		base="$$(basename $$src)"; \
		dst="rules/_schema/catalog.$$base"; \
		cp -p "$$src" "$$dst" && echo "synced $$src -> $$dst"; \
	done

up: ## Start the local docker-compose substrate (per ADR-0010 Yes capabilities).
	@$(COMPOSE) up -d --wait

down: ## Stop the local docker-compose substrate.
	@$(COMPOSE) down --remove-orphans

smoke-substrate: ## Run the four substrate smoke tests against the running local Compose stack.
	@bash scripts/smoke/pubsub-smoke.sh
	@bash scripts/smoke/object-store-smoke.sh
	@bash scripts/smoke/tabular-store-smoke.sh
	@bash scripts/smoke/event-stream-smoke.sh

demo-p6: ## End-to-end Phase 6 demo (W3-P6d). Closes the W2-3 C-W2-3.4 invariant locally. Requires `make up` first.
	@bash scripts/smoke/demo-p6.sh

# B2-8 CC2 names `kubectl apply -k --dry-run=client` as the validation
# lane. Empirically that command still performs API-server discovery
# before parsing — `unable to recognize` fails on any cluster-free
# host (verified with kubectl 1.36 against an empty kubeconfig).
# `kubectl kustomize` is the maximum-portable cluster-free render
# surface; B2-30 (per ADR-0042 Clause 4) layered the deeper-
# validation lane (`kubeconform -strict`) into the same target so
# field-name typos (e.g. `replicass:`), deprecated API versions,
# and schema mismatches against the Kubernetes API surface fail
# loudly without introducing a new top-level CI lane.
validate-deploy: ## Render every overlay via `kubectl kustomize` + validate via `kubeconform -strict` (B2-8 CC2/CC7 + ADR-0042 Clause 4).
	@if ! command -v kubeconform >/dev/null 2>&1; then \
		echo "validate-deploy: kubeconform not on PATH" >&2; \
		echo "  install via 'brew install kubeconform' (macOS), or download a release from https://github.com/yannh/kubeconform/releases" >&2; \
		exit 1; \
	fi
	@set -e; \
	for env in local qa prod; do \
		echo "→ validating deploy/overlays/$$env/"; \
		kubectl kustomize deploy/overlays/$$env/ \
			| kubeconform -strict -summary -; \
	done
