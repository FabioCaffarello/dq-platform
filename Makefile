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

lint-rules: build-lint ## Validate every rule YAML against the schema mirror (per ADR-0001).
	@./bin/dq-lint -schema rules/_schema/v1.schema.json -rules rules

dry-run-rules: ## Stub: generate SQL for every rule without executing (lands in Phase 4 with tools/dryrun).
	@echo "dry-run-rules: not yet implemented; lands in Phase 4 (tools/dryrun binary)."

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
# `kubectl kustomize` is the maximum-portable cluster-free surface of
# CC2's intent: it catches YAML syntax errors, missing-resource
# references, patch-target mismatches, and strategic-merge conflicts.
# Deeper schema validation (field-name typos, e.g. `replicass:`) is a
# follow-up lane — `kubeconform` or a kind-based CI cluster —
# deferred to B2-3 release-engineering or a Phase-7 follow-up.
validate-deploy: ## Render every overlay via `kubectl kustomize` (per B2-8 CC2/CC7).
	@set -e; \
	for env in local qa prod; do \
		echo "→ rendering deploy/overlays/$$env/"; \
		kubectl kustomize deploy/overlays/$$env/ > /dev/null; \
	done
