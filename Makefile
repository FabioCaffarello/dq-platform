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
	lint-engine test-engine \
	lint-rules dry-run-rules \
	up down \
	smoke-substrate

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z][a-zA-Z0-9_-]+:.*?## / { printf "  \033[1m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

lint: lint-engine ## Lint everything that changed since main (currently: engine; rules-lint stub).
	@echo "lint: rules-lint deferred — lands in Phase 3 (tools/lint binary)."

test: smoke-substrate test-engine ## Run substrate smoke tests + go tests across modules.

lint-engine: ## go vet across the engine module.
	@cd engine && go vet ./...

test-engine: ## go test across the engine module.
	@cd engine && go test ./...

lint-rules: ## Stub: validate every rule YAML (lands in Phase 3 with tools/lint).
	@echo "lint-rules: not yet implemented; lands in Phase 3 (tools/lint binary)."

dry-run-rules: ## Stub: generate SQL for every rule without executing (lands in Phase 4 with tools/dryrun).
	@echo "dry-run-rules: not yet implemented; lands in Phase 4 (tools/dryrun binary)."

up: ## Start the local docker-compose substrate (per ADR-0010 Yes capabilities).
	@$(COMPOSE) up -d --wait

down: ## Stop the local docker-compose substrate.
	@$(COMPOSE) down --remove-orphans

smoke-substrate: ## Run the three substrate smoke tests against the running local Compose stack.
	@bash scripts/smoke/pubsub-smoke.sh
	@bash scripts/smoke/object-store-smoke.sh
	@bash scripts/smoke/tabular-store-smoke.sh
