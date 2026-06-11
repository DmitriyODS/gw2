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
	@printf "  make dev-infra    Инфра в Docker: DB + Redis + LiveKit\n"
	@printf "  make dev-calls    Go-микросервис звонков (gRPC :9090, HTTP :8090)\n"
	@printf "  make dev-back     Flask dev-сервер :5001\n"
	@printf "  make dev-front    Vite dev-сервер  :5173\n"
	@printf "  make dev-stop     Остановить dev-контейнеры\n"
	@printf "  make dev-stack    ВЕСЬ стек в Docker (прод-подобно, фронт :8080)\n"
	@printf "  make dev-stack-stop  Остановить полный стек\n"
	@printf "  make gen-proto    Перегенерировать gRPC-стабы (Go + Python)\n"
	@printf "\n\033[1mДеплой (сервер):\033[0m\n"
	@printf "  make deploy       git push → fetch+reset на сервере → docker compose up --build\n"
	@printf "  make logs [s=calls]     Логи контейнера (по умолчанию app)\n"
	@printf "  make status       docker compose ps на сервере\n"
	@printf "  make restart [s=calls]  Перезапустить контейнер без пересборки\n"
	@printf "  make shell [s=calls]    Шелл внутри контейнера на сервере\n"
	@printf "  make reset NEWPASS='...'  Сбросить пароль суперадмина на сервере\n"
	@printf "\n\033[33mКонфигурация сервера:\033[0m cp .env.deploy.example .env.deploy\n\n"

# ── Разработка ────────────────────────────────────────────────────
.PHONY: dev-infra dev-migrate dev-back dev-front dev-calls dev-stop dev-stack dev-stack-stop gen-proto

# Приложения (app/calls/nginx) в dev-оверлее за профилем "full" —
# bare `up` поднимает только инфраструктуру.
dev-infra:
	@printf "\033[1m▶ Запускаю DB + Redis + LiveKit...\033[0m\n"
	cd deploy && docker compose up -d
	@printf "\033[32m✓ PostgreSQL :5432  Redis :6379  LiveKit :7880\033[0m\n"

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

# Go-микросервис звонков: бизнес-логика, LiveKit, REST /api/calls/* и gRPC
# для Flask-шлюза. env синхронизированы с back/.flaskenv и dev.sh.
dev-calls: dev-infra
	@printf "\033[1m▶ callsvc (Go)  gRPC :9090  HTTP :8090\033[0m\n"
	cd back-go/calls && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	JWT_SECRET_KEY="dev-jwt-secret-key-min-32-chars-local-xxxx" \
	LIVEKIT_API_KEY="devkey" \
	LIVEKIT_API_SECRET="dev_livekit_secret_min_32_chars_ok" \
	LIVEKIT_URL="http://localhost:7880" \
	LIVEKIT_CLIENT_URL="ws://localhost:7880" \
	go run ./cmd/callsvc

gen-proto:
	bash scripts/gen_proto.sh

dev-stop:
	cd deploy && docker compose --profile full stop
	@printf "\033[32m✓ Dev-контейнеры остановлены\033[0m\n"

# Полный стек в контейнерах — прод-подобная проверка сборки/композиции
# (Flask, callsvc и nginx из образов; фронт на http://localhost:8080).
dev-stack:
	@printf "\033[1m▶ Полный стек в Docker...\033[0m\n"
	cd deploy && docker compose --profile full up -d --build
	@printf "\033[32m✓ Фронт http://localhost:8080  API http://localhost:8080/api\033[0m\n"

dev-stack-stop:
	cd deploy && docker compose --profile full stop
	@printf "\033[32m✓ Полный стек остановлен\033[0m\n"

# ── Деплой ───────────────────────────────────────────────────────
.PHONY: deploy logs status restart shell

# Прод-стек = база + оверлей (см. шапку deploy/docker-compose.prod.yml).
COMPOSE_PROD := docker compose -f docker-compose.yml -f docker-compose.prod.yml
# Сервис для logs/restart/shell: make logs s=calls (по умолчанию app).
s ?= app

deploy:
	@printf "\033[1m▶ Пушу в GitHub...\033[0m\n"
	git push
	@printf "\033[1m▶ Деплою на $(SERVER_HOST)...\033[0m\n"
	@# Вся серверная логика — в scripts/deploy_server.sh (приезжает тем же
	@# git reset): синк .env с автогенерацией недостающих секретов, firewall,
	@# up --build --remove-orphans, reload nginx, health-чеки.
	$(SSH) "cd $(SERVER_DIR) && git fetch origin && git reset --hard origin/main && bash scripts/deploy_server.sh"
	@printf "\033[32m✓ Задеплоено на $(SERVER_HOST)\033[0m\n"

logs:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) logs -f --tail=200 $(s)"

status:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) ps"

restart:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) restart $(s)"
	@printf "\033[32m✓ $(s) перезапущен\033[0m\n"

shell:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) exec $(s) sh -c 'command -v bash >/dev/null && exec bash || exec sh'"

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
