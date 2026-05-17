# ================================================================
# Grove Work — автоматизация разработки и деплоя
# ================================================================
# Конфигурация сервера: cp .env.deploy.example .env.deploy
# ================================================================
-include .env.deploy

SERVER_USER ?= root
SERVER_HOST ?= CONFIGURE_IN_.ENV.DEPLOY
SERVER_DIR  ?= /opt/gw2
SSH_KEY     ?= ~/.ssh/id_rsa
SSH         := ssh -i $(SSH_KEY) $(SERVER_USER)@$(SERVER_HOST)

.DEFAULT_GOAL := help

# ── Справка ──────────────────────────────────────────────────────
.PHONY: help
help:
	@printf "\n\033[1mGrove Work — доступные команды\033[0m\n"
	@printf "\n\033[1mРазработка (локально):\033[0m\n"
	@printf "  make dev-infra    Запустить DB + Redis в Docker\n"
	@printf "  make dev-back     Flask dev-сервер :5001\n"
	@printf "  make dev-front    Vite dev-сервер  :5173\n"
	@printf "  make dev-stop     Остановить DB + Redis\n"
	@printf "\n\033[1mДеплой (сервер):\033[0m\n"
	@printf "  make deploy       git push → git pull на сервере → docker compose up --build\n"
	@printf "  make logs         Стримить логи app-контейнера (Ctrl+C выйти)\n"
	@printf "  make status       docker compose ps на сервере\n"
	@printf "  make restart      Перезапустить app без пересборки\n"
	@printf "  make shell        bash внутри app-контейнера на сервере\n"
	@printf "\n\033[33mКонфигурация сервера:\033[0m cp .env.deploy.example .env.deploy\n\n"

# ── Разработка ────────────────────────────────────────────────────
.PHONY: dev-infra dev-migrate dev-back dev-front dev-stop

dev-infra:
	@printf "\033[1m▶ Запускаю DB + Redis...\033[0m\n"
	cd deploy && docker compose up -d db redis
	@printf "\033[32m✓ PostgreSQL :5432  Redis :6379\033[0m\n"

dev-migrate: dev-infra
	@printf "\033[1m▶ Применяю миграции...\033[0m\n"
	cd back && . venv/bin/activate && flask db upgrade
	@printf "\033[32m✓ Миграции применены\033[0m\n"

dev-back: dev-migrate
	@printf "\033[1m▶ Flask :5001\033[0m\n"
	cd back && . venv/bin/activate && flask run --debug --port 5001

dev-front:
	@printf "\033[1m▶ Vite :5173\033[0m\n"
	cd front && npm run dev

dev-stop:
	cd deploy && docker compose stop db redis
	@printf "\033[32m✓ DB и Redis остановлены\033[0m\n"

# ── Деплой ───────────────────────────────────────────────────────
.PHONY: deploy logs status restart shell

deploy:
	@printf "\033[1m▶ Пушу в GitHub...\033[0m\n"
	git push
	@printf "\033[1m▶ Деплою на $(SERVER_HOST)...\033[0m\n"
	$(SSH) "cd $(SERVER_DIR) && git pull && cd deploy && docker compose up -d --build"
	@printf "\033[32m✓ Задеплоено на $(SERVER_HOST)\033[0m\n"

logs:
	$(SSH) "cd $(SERVER_DIR)/deploy && docker compose logs -f --tail=200 app"

status:
	$(SSH) "cd $(SERVER_DIR)/deploy && docker compose ps"

restart:
	$(SSH) "cd $(SERVER_DIR)/deploy && docker compose restart app"
	@printf "\033[32m✓ app перезапущен\033[0m\n"

shell:
	$(SSH) "cd $(SERVER_DIR)/deploy && docker compose exec app bash"
