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
	lint-tools test-tools \
	lint-rules dry-run-rules \
	sync-schema \
	build-lint \
	up down \
	smoke-substrate

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

test-tools: ## go test across every module under tools/.
	@cd tools/lint && go test ./...

build-lint: ## Build the dq-lint binary at bin/dq-lint.
	@mkdir -p bin
	@cd tools/lint && go build -o ../../bin/dq-lint .

lint-rules: build-lint ## Validate every rule YAML against the schema mirror (per ADR-0001).
	@./bin/dq-lint -schema rules/_schema/v1.schema.json -rules rules

dry-run-rules: ## Stub: generate SQL for every rule without executing (lands in Phase 4 with tools/dryrun).
	@echo "dry-run-rules: not yet implemented; lands in Phase 4 (tools/dryrun binary)."

sync-schema: ## Mechanically derive the rules schema mirror from the engine source (ADR-0001 C3).
	@for src in engine/internal/dsl/schema/v*.schema.json; do \
		base="$$(basename $$src)"; \
		dst="rules/_schema/$$base"; \
		cp -p "$$src" "$$dst" && echo "synced $$src -> $$dst"; \
	done

up: ## Start the local docker-compose substrate (per ADR-0010 Yes capabilities).
	@$(COMPOSE) up -d --wait

down: ## Stop the local docker-compose substrate.
	@$(COMPOSE) down --remove-orphans

smoke-substrate: ## Run the three substrate smoke tests against the running local Compose stack.
	@bash scripts/smoke/pubsub-smoke.sh
	@bash scripts/smoke/object-store-smoke.sh
	@bash scripts/smoke/tabular-store-smoke.sh
