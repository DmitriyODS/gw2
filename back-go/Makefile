SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

# Пин-версии тулзов (для воспроизводимости)
STATICCHECK_VERSION  := 2025.1.1
GOVULNCHECK_VERSION  := latest

BIN_DIR  := $(CURDIR)/bin
SERVICES := auth users tasks messenger calls ai

# Цвета
C_RESET := \033[0m
C_BOLD  := \033[1m
C_BLUE  := \033[34m
C_GREEN := \033[32m

.PHONY: help
help: ## Показать доступные таргеты
	@printf "$(C_BOLD)Groove Work v4 — back-go targets$(C_RESET)\n\n"
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_\/.-]+:.*## / {printf "  $(C_BLUE)%-22s$(C_RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\n"

# ───────── build / run ─────────

.PHONY: build
build: $(addprefix build/,$(SERVICES)) ## Собрать все сервисы в bin/

.PHONY: build/%
build/%: ## Собрать один сервис (build/auth)
	@mkdir -p $(BIN_DIR)
	@printf "$(C_GREEN)→ building $*$(C_RESET)\n"
	@cd services/$* && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
		-o $(BIN_DIR)/$* ./cmd/$*

.PHONY: run/%
run/%: ## Запустить сервис локально (run/auth)
	@cd services/$* && go run ./cmd/$*

.PHONY: clean
clean: ## Удалить bin/ и кэш тестов
	@rm -rf $(BIN_DIR)
	@go clean -testcache

# ───────── tests ─────────

.PHONY: test
test: ## Unit-тесты во всех модулях
	@for SVC in $(SERVICES); do \
		printf "$(C_GREEN)→ unit $$SVC$(C_RESET)\n"; \
		(cd services/$$SVC && go test -short -race -count=1 ./...) || exit 1; \
	done
	@printf "$(C_GREEN)→ unit shared/pkg$(C_RESET)\n"
	@cd shared/pkg && go test -short -race -count=1 ./...

.PHONY: test/integration
test/integration: ## Integration (testcontainers, нужен Docker)
	@for SVC in $(SERVICES); do \
		printf "$(C_GREEN)→ integration $$SVC$(C_RESET)\n"; \
		(cd services/$$SVC && go test -tags=integration -race -count=1 -timeout=10m ./...) || exit 1; \
	done

.PHONY: test/e2e
test/e2e: ## E2E (testcontainers, нужен Docker)
	@for SVC in $(SERVICES); do \
		printf "$(C_GREEN)→ e2e $$SVC$(C_RESET)\n"; \
		(cd services/$$SVC && go test -tags=e2e -race -count=1 -timeout=10m ./...) || exit 1; \
	done

.PHONY: test/all
test/all: test test/integration test/e2e ## Все уровни тестов

# ───────── quality gates ─────────

.PHONY: fmt
fmt: ## gofmt -s по всем модулям
	@gofmt -w -s services shared gen

.PHONY: fmt/check
fmt/check: ## Проверить, что весь код отформатирован
	@diff=$$(gofmt -d -s services shared gen); \
	if [ -n "$$diff" ]; then \
		printf "$(C_BOLD)unformatted:$(C_RESET)\n%s\n" "$$diff"; exit 1; \
	fi

.PHONY: vet
vet: ## go vet во всех модулях
	@for SVC in $(SERVICES); do \
		(cd services/$$SVC && go vet ./...) || exit 1; \
	done
	@cd shared/pkg && go vet ./...
	@cd gen/proto && go vet ./...

.PHONY: tidy
tidy: ## go mod tidy во всех модулях
	@cd shared/pkg && go mod tidy
	@cd gen/proto && go mod tidy
	@for SVC in $(SERVICES); do \
		(cd services/$$SVC && go mod tidy) || exit 1; \
	done

.PHONY: tidy/check
tidy/check: ## go mod tidy -diff во всех модулях
	@cd shared/pkg && go mod tidy -diff
	@cd gen/proto && go mod tidy -diff
	@for SVC in $(SERVICES); do \
		(cd services/$$SVC && go mod tidy -diff) || exit 1; \
	done

.PHONY: verify
verify: ## go mod verify во всех модулях
	@cd shared/pkg && go mod verify
	@cd gen/proto && go mod verify
	@for SVC in $(SERVICES); do \
		(cd services/$$SVC && go mod verify) || exit 1; \
	done

.PHONY: staticcheck
staticcheck: ## staticcheck во всех модулях
	@for SVC in $(SERVICES); do \
		(cd services/$$SVC && go run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) ./...) || exit 1; \
	done

.PHONY: vulncheck
vulncheck: ## govulncheck во всех модулях
	@for SVC in $(SERVICES); do \
		(cd services/$$SVC && go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...) || exit 1; \
	done

.PHONY: lint
lint: fmt/check vet staticcheck ## Все статические проверки

.PHONY: audit
audit: tidy/check verify lint vulncheck test ## Полный CI-аудит (без integration/e2e)

# ───────── proto / codegen ─────────

.PHONY: gen/proto
gen/proto: ## Перегенерировать gRPC-код через buf
	@printf "$(C_GREEN)→ buf generate$(C_RESET)\n"
	@buf generate
	@cd gen/proto && go mod tidy

.PHONY: gen/proto/lint
gen/proto/lint: ## buf lint
	@buf lint

.PHONY: gen/proto/breaking
gen/proto/breaking: ## buf breaking против main
	@buf breaking --against '.git#branch=main'
