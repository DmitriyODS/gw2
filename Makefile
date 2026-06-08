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
	@printf "  make deploy       git push → fetch+reset на сервере → docker compose up --build\n"
	@printf "  make logs         Стримить логи app-контейнера (Ctrl+C выйти)\n"
	@printf "  make status       docker compose ps на сервере\n"
	@printf "  make restart      Перезапустить app без пересборки\n"
	@printf "  make shell        bash внутри app-контейнера на сервере\n"
	@printf "  make reset NEWPASS='...'  Сбросить пароль суперадмина на сервере\n"
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
	@printf "\033[1m▶ Flask + eventlet :5001\033[0m\n"
	@# Запускаем через wsgi.py (eventlet). Werkzeug-сервер из flask run
	@# не поддерживает WebSocket — поэтому socket.io WS-upgrade на нём фейлится.
	@# Auto-reload в dev отсутствует: перезапускайте процесс после изменений.
	cd back && . venv/bin/activate && PORT=5001 python wsgi.py

dev-front:
	@printf "\033[1m▶ Vite :5173\033[0m\n"
	cd front && npm run dev

dev-stop:
	cd deploy && docker compose stop db redis
	@printf "\033[32m✓ DB и Redis остановлены\033[0m\n"

# ── Деплой ───────────────────────────────────────────────────────
.PHONY: deploy logs status restart shell

COMPOSE_PROD := docker compose -f docker-compose.prod.yml

deploy:
	@printf "\033[1m▶ Пушу в GitHub...\033[0m\n"
	git push
	@printf "\033[1m▶ Деплою на $(SERVER_HOST)...\033[0m\n"
	$(SSH) "cd $(SERVER_DIR) && git fetch origin && git reset --hard origin/main && cd deploy && $(COMPOSE_PROD) up -d --build"
	@printf "\033[32m✓ Задеплоено на $(SERVER_HOST)\033[0m\n"

logs:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) logs -f --tail=200 app"

status:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) ps"

restart:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) restart app"
	@printf "\033[32m✓ app перезапущен\033[0m\n"

shell:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) exec app bash"

# ── Сброс пароля суперадмина ─────────────────────────────────────
# Использование: make reset NEWPASS='новый-пароль'
# Меняет hash_password у системного суперадмина (минимальный id среди
# пользователей с role.level=4) и сбрасывает is_default_pass=FALSE.
.PHONY: reset
reset:
	@if [ -z "$(NEWPASS)" ]; then \
		printf "\033[31m✗ Передайте новый пароль:\033[0m  make reset NEWPASS='новый-пароль'\n"; \
		exit 2; \
	fi
	@printf "\033[1m▶ Сбрасываю пароль суперадмина на $(SERVER_HOST)...\033[0m\n"
	@scripts/reset_superadmin_password.sh "$(SERVER_USER)@$(SERVER_HOST)" "$(SSH_KEY)" "$(SERVER_DIR)" "$(NEWPASS)"
