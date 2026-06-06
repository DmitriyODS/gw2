SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

COMPOSE := docker compose -f docker-compose.yml --env-file .env
SERVICES := auth users tasks messenger calls ai
INFRA := postgres redis mailhog

# Цвет
C_RESET := \033[0m
C_BOLD  := \033[1m
C_BLUE  := \033[34m
C_GREEN := \033[32m

.PHONY: help
help: ## Показать доступные таргеты
	@printf "$(C_BOLD)Groove Work v4 — deploy-v2 targets$(C_RESET)\n\n"
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_\/.-]+:.*## / {printf "  $(C_BLUE)%-22s$(C_RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\n"

.PHONY: .env
.env: ## Создать .env из .env.example если ещё нет
	@if [ ! -f .env ]; then cp .env.example .env && printf "$(C_GREEN)→ создан .env (отредактируйте при необходимости)$(C_RESET)\n"; fi

# ───────── dev ─────────

.PHONY: dev/up
dev/up: .env ## Поднять весь стек (postgres + redis + mailhog + 6 сервисов)
	@$(COMPOSE) up -d --build

.PHONY: dev/down
dev/down: ## Остановить стек (тома сохраняются)
	@$(COMPOSE) down

.PHONY: dev/fresh
dev/fresh: ## Снести вместе с томами (потеря dev-данных)
	@$(COMPOSE) down -v --remove-orphans

.PHONY: dev/ps
dev/ps: ## Статус контейнеров
	@$(COMPOSE) ps

.PHONY: dev/logs
dev/logs: ## Логи всего стека (tail -f)
	@$(COMPOSE) logs -f --tail=200

.PHONY: dev/logs/%
dev/logs/%: ## Логи одного сервиса (dev/logs/auth)
	@$(COMPOSE) logs -f --tail=200 $*

.PHONY: dev/restart/%
dev/restart/%: ## Перезапустить один сервис (dev/restart/auth)
	@$(COMPOSE) restart $*

.PHONY: dev/rebuild/%
dev/rebuild/%: ## Пересобрать и перезапустить один сервис (dev/rebuild/auth)
	@$(COMPOSE) up -d --build --force-recreate $*

.PHONY: dev/infra
dev/infra: .env ## Поднять ТОЛЬКО инфру (для локального запуска сервисов без docker)
	@$(COMPOSE) up -d $(INFRA)

# ───────── БД ─────────

.PHONY: dev/psql
dev/psql: ## psql внутрь postgres
	@$(COMPOSE) exec postgres psql -U $${POSTGRES_USER:-grovework} -d $${POSTGRES_DB:-grovework_v4}

# ───────── проверки ─────────

.PHONY: dev/health
dev/health: ## Опросить /v1/health всех 6 сервисов
	@for SVC in $(SERVICES); do \
		case $$SVC in \
			auth) PORT=8081 ;; \
			users) PORT=8082 ;; \
			tasks) PORT=8083 ;; \
			messenger) PORT=8084 ;; \
			calls) PORT=8085 ;; \
			ai) PORT=8086 ;; \
		esac; \
		printf "  $(C_BLUE)%-12s$(C_RESET) " $$SVC; \
		curl -sf -m 2 http://localhost:$$PORT/v1/health 2>/dev/null || echo "DOWN"; \
		echo; \
	done

.PHONY: validate
validate: ## Проверить, что compose-файлы валидны
	@$(COMPOSE) config -q && printf "$(C_GREEN)→ compose валиден$(C_RESET)\n"
