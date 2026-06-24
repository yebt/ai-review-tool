SHELL := /bin/bash

.DEFAULT_GOAL := help

WEB_SPA_DIR := packages/web-spa
SERVER_DIR := packages/server
BUN := bun
GO := go

.PHONY: help check install dev type-check test build lint format preview \
	spa-check spa-install spa-dev spa-type-check spa-test spa-build spa-lint spa-format spa-preview \
	server-test server-run server-build

help: ## Show this help message.
	@printf '\033[1;36mCo-Review workspace commands\033[0m\n\n'
	@printf '\033[1;33mUsage:\033[0m make <target>\n\n'
	@printf '\033[1;34mWorkspace shortcuts\033[0m\n'
	@printf '  \033[32m%-18s\033[0m %s\n' 'help' 'Show this help message.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'install' 'Install dependencies for available apps.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'dev' 'Run the SPA development server.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'check' 'Run type-check, unit tests, and build for available apps.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'type-check' 'Run type-check for available apps.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'test' 'Run unit tests for available apps.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'build' 'Build available apps.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'lint' 'Run lint/fix for available apps.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'format' 'Format available apps.'
	@printf '  \033[32m%-18s\033[0m %s\n\n' 'preview' 'Preview the SPA production build.'
	@printf '\033[1;34mSPA: $(WEB_SPA_DIR)\033[0m\n'
	@printf '  \033[32m%-18s\033[0m %s\n' 'spa-install' 'Run bun install.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'spa-dev' 'Run bun run dev.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'spa-check' 'Run SPA type-check, unit tests, and build.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'spa-type-check' 'Run bun run type-check.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'spa-test' 'Run bun run test:unit.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'spa-build' 'Run bun run build.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'spa-lint' 'Run bun run lint.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'spa-format' 'Run bun run format.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'spa-preview' 'Run bun run preview.'
	@printf '\n\033[1;34mServer: $(SERVER_DIR)\033[0m\n'
	@printf '  \033[32m%-18s\033[0m %s\n' 'server-test' 'Run go test ./... from the server package.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'server-build' 'Build the server binary.'
	@printf '  \033[32m%-18s\033[0m %s\n' 'server-run' 'Run the server locally.'

install: spa-install ## Install dependencies for available apps.

dev: spa-dev ## Run the SPA development server.

check: spa-check server-test server-build ## Run type-check, unit tests, and build for available apps.

type-check: spa-type-check ## Run type-check for available apps.

test: spa-test server-test ## Run unit tests for available apps.

build: spa-build server-build ## Build available apps.

lint: spa-lint ## Run lint/fix for available apps.

format: spa-format ## Format available apps.

preview: spa-preview ## Preview the SPA production build.

spa-check: spa-type-check spa-test spa-build ## Run SPA type-check, unit tests, and build.

spa-install: ## Install SPA dependencies.
	@printf '\033[1;35m==> Installing SPA dependencies\033[0m\n'
	@cd $(WEB_SPA_DIR) && $(BUN) install

spa-dev: ## Run SPA development server.
	@printf '\033[1;35m==> Starting SPA development server\033[0m\n'
	@cd $(WEB_SPA_DIR) && $(BUN) run dev

spa-type-check: ## Run SPA type-check.
	@printf '\033[1;35m==> Type-checking SPA\033[0m\n'
	@cd $(WEB_SPA_DIR) && $(BUN) run type-check

spa-test: ## Run SPA unit tests.
	@printf '\033[1;35m==> Testing SPA\033[0m\n'
	@cd $(WEB_SPA_DIR) && $(BUN) run test:unit

spa-build: ## Build SPA.
	@printf '\033[1;35m==> Building SPA\033[0m\n'
	@cd $(WEB_SPA_DIR) && $(BUN) run build

spa-lint: ## Run SPA lint/fix.
	@printf '\033[1;35m==> Linting SPA\033[0m\n'
	@cd $(WEB_SPA_DIR) && $(BUN) run lint

spa-format: ## Format SPA source.
	@printf '\033[1;35m==> Formatting SPA\033[0m\n'
	@cd $(WEB_SPA_DIR) && $(BUN) run format

spa-preview: ## Preview SPA production build.
	@printf '\033[1;35m==> Previewing SPA production build\033[0m\n'
	@cd $(WEB_SPA_DIR) && $(BUN) run preview

server-test: ## Test server package.
	@printf '\033[1;35m==> Testing server\033[0m\n'
	@cd $(SERVER_DIR) && $(GO) test ./...

server-build: ## Build server package.
	@printf '\033[1;35m==> Building server\033[0m\n'
	@cd $(SERVER_DIR) && $(GO) build -o bin/server ./cmd/server

server-run: ## Run server package.
	@printf '\033[1;35m==> Starting server\033[0m\n'
	@cd $(SERVER_DIR) && $(GO) run ./cmd/server
