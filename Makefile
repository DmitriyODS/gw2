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
	@printf "  make dev-auth     Go-микросервис авторизации (HTTP :8091)\n"
	@printf "  make dev-messenger  Go-микросервис мессенджера (gRPC :9092, HTTP :8092)\n"
	@printf "  make dev-ai       Go-микросервис ИИ (gRPC :9093, HTTP :8093)\n"
	@printf "  make dev-pets     Go-микросервис питомцев-грувиков (gRPC :9094, HTTP :8094)\n"
	@printf "  make dev-tasks    Go-микросервис задач (HTTP :8095)\n"
	@printf "  make dev-gateway  Realtime-шлюз (WS /ws, HTTP :8096)\n"
	@printf "  make dev-push     Go-микросервис пуш-уведомлений (HTTP :8097)\n"
	@printf "  make dev-mail     Go-микросервис почты (gRPC :9098, HTTP :8098)\n"
	@printf "  make dev-registry Go-микросервис реестров (HTTP :8099)\n"
	@printf "  make dev-calendar Go-микросервис календарей (HTTP :8100)\n"
	@printf "  make dev-diary    Go-микросервис ежедневников (HTTP :8101)\n"
	@printf "  make dev-portal   Go-микросервис корпоративного портала (HTTP :8102)\n"
	@printf "  make dev-notes    Go-микросервис заметок (HTTP :8103)\n"
	@printf "  make dev-migrate  Применить миграции (goose)\n"
	@printf "  make dev-front    Vite dev-сервер  :5173\n"
	@printf "  make dev-stop     Остановить dev-контейнеры\n"
	@printf "  make dev-stack    ВЕСЬ стек в Docker (прод-подобно, фронт :8080)\n"
	@printf "  make dev-stack-stop  Остановить полный стек\n"
	@printf "  make gen-proto    Перегенерировать gRPC-стабы (Go + Python)\n"
	@printf "\n\033[1mДеплой (сервер):\033[0m\n"
	@printf "  make push [only=\"gateway front\"]  Собрать (linux/amd64) и запушить изменившиеся образы\n"
	@printf "  make push-all     Принудительно пересобрать и запушить ВСЕ образы\n"
	@printf "  make deploy       make push → git push → на сервере: compose pull + up --no-build\n"
	@printf "  make deploy-only  То же без сборки/пуша образов (push уже сделан)\n"
	@printf "  make apk          Собрать мобильное приложение (Capacitor) → apps/mobile/ + зеркало apps/\n"
	@printf "  make deploy-apk   Залить APK и version.json обоих каналов на сервер\n"
	@printf "  make desktop      Собрать десктоп-клиент (dmg+exe+AppImage) → apps/desktop/ (версия: V=1.0.1)\n"
	@printf "  make deploy-desktop  Залить десктоп-клиент из apps/desktop/ на сервер\n"
	@printf "  make release MSG=\"...\" [V=1.0.6] [BUILD=2607111]  ПОЛНЫЙ релиз: коммит, версии+сборка обоих приложений, деплой сервера и артефактов\n"
	@printf "  make logs [s=calls]     Логи контейнера (по умолчанию gateway)\n"
	@printf "  make status       docker compose ps на сервере\n"
	@printf "  make restart [s=calls]  Перезапустить контейнер без пересборки\n"
	@printf "  make shell [s=calls]    Шелл внутри контейнера на сервере\n"
	@printf "  make backup       Дамп прод-БД → backups/gw2_<дата>.sql.gz (локально)\n"
	@printf "  make reset NEWPASS='...'  Сбросить пароль суперадмина на сервере\n"
	@printf "  make dev-reset NEWPASS='...'  Сбросить пароль суперадмина в dev-БД\n"
	@printf "  make dev-seed             Залить демо-данные в dev-БД (demo.* / demo1234)\n"
	@printf "\n\033[33mКонфигурация сервера:\033[0m cp .env.deploy.example .env.deploy\n\n"

# ── Разработка ────────────────────────────────────────────────────
.PHONY: dev-infra dev-migrate dev-front dev-calls dev-auth dev-messenger dev-ai dev-pets dev-tasks dev-gateway dev-push dev-mail dev-registry dev-calendar dev-diary dev-portal dev-notes dev-stop dev-stack dev-stack-stop gen-proto

# Dev-ключи PASETO (синхронизированы с dev.sh и
# deploy/docker-compose.override.yml): приватный — только у authsvc,
# публичный — у остальных сервисов.
PASETO_PRIVATE_KEY_DEV := 68eb779b2f672beb8fcd58d72a81ce1565a1417aed3788d1362bf4faaa3f62ac15ef439747fcad6ca627310942ba14b48f164fcbb5f65c10f61ca2aeb4b53fe1
PASETO_PUBLIC_KEY_DEV  := 15ef439747fcad6ca627310942ba14b48f164fcbb5f65c10f61ca2aeb4b53fe1
PASETO_REFRESH_KEY_DEV := d525374c4ec7b5e1c5b140fb9c1f4cffd9c3dbf052bb18f2f32bf9f92d9fa05c

# Приложения в dev-оверлее за профилем "full" — bare `up` поднимает
# только инфраструктуру.
dev-infra:
	@printf "\033[1m▶ Запускаю DB + Redis + LiveKit...\033[0m\n"
	cd deploy && docker compose up -d
	@printf "\033[32m✓ PostgreSQL :5432  Redis :6379  LiveKit :7880\033[0m\n"

dev-migrate: dev-infra
	@printf "\033[1m▶ Применяю миграции (goose)...\033[0m\n"
	cd back-go/migrate && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	go run ./cmd/migrate
	@printf "\033[32m✓ Миграции применены\033[0m\n"

dev-front:
	@printf "\033[1m▶ Vite :5173\033[0m\n"
	cd front && npm run dev

# Go-микросервис звонков: бизнес-логика, LiveKit, REST /api/calls/* и gRPC
# ринг-фазы (зовёт gateway); плашки звонков — gRPC msgsvc.
# env синхронизированы с dev.sh.
dev-calls: dev-infra
	@printf "\033[1m▶ callsvc (Go)  gRPC :9090  HTTP :8090\033[0m\n"
	cd back-go/calls && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	LIVEKIT_API_KEY="devkey" \
	LIVEKIT_API_SECRET="dev_livekit_secret_min_32_chars_ok" \
	LIVEKIT_URL="http://localhost:7880" \
	LIVEKIT_CLIENT_URL="ws://localhost:7880" \
	MESSENGER_GRPC_ADDR="localhost:9092" \
	go run ./cmd/callsvc

# Go-микросервис авторизации: /api/auth/* и /api/users/*, выпуск PASETO-токенов
# (access v4.public + refresh v4.local). env синхронизированы с dev.sh.
dev-auth: dev-infra
	@printf "\033[1m▶ authsvc (Go)  HTTP :8091\033[0m\n"
	cd back-go/auth && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PRIVATE_KEY="$(PASETO_PRIVATE_KEY_DEV)" \
	PASETO_REFRESH_KEY="$(PASETO_REFRESH_KEY_DEV)" \
	UPLOAD_FOLDER="$(CURDIR)/uploads" \
	MAIL_GRPC_ADDR="localhost:9098" \
	APP_PUBLIC_BASE_URL="http://localhost:5173" \
	go run ./cmd/authsvc

# Go-микросервис мессенджера: REST /api/messenger/* (кроме exact presence —
# он в gateway) и gRPC (плашки звонков, pet-чат). env синхронизированы с dev.sh.
dev-messenger: dev-infra
	@printf "\033[1m▶ msgsvc (Go)  gRPC :9092  HTTP :8092\033[0m\n"
	cd back-go/messenger && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	UPLOAD_FOLDER="$(CURDIR)/uploads" \
	HTTP_ADDR=":8092" \
	GRPC_ADDR=":9092" \
	go run ./cmd/msgsvc

# Go-микросервис ИИ: REST /api/companies/<id>/ai-settings (regex-роут в
# nginx/vite) + /api/ai/tv-fact и gRPC для tasksvc. Redis — кэш
# ТВ-фактов. env синхронизированы с dev.sh.
dev-ai: dev-infra
	@printf "\033[1m▶ aisvc (Go)  gRPC :9093  HTTP :8093\033[0m\n"
	cd back-go/ai && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	AI_KEY_ENCRYPTION_KEY="X3hFOVZ6XbAzlaygv2PfLbnmBIaH373CK8MqrrAhr8k=" \
	HTTP_ADDR=":8093" \
	GRPC_ADDR=":9093" \
	go run ./cmd/aisvc

# Go-микросервис питомцев-грувиков: REST /api/pets/* и gRPC-хуки доменных
# событий (tasksvc — юниты/задачи). Исходящих межсервисных вызовов нет.
# env синхронизированы с dev.sh.
dev-pets: dev-infra
	@printf "\033[1m▶ petsvc (Go)  gRPC :9094  HTTP :8094\033[0m\n"
	cd back-go/pets && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	HTTP_ADDR=":8094" \
	GRPC_ADDR=":9094" \
	go run ./cmd/petsvc

# Go-микросервис задач: ядро платформы — REST /api/tasks|units|unit-types|
# departments|stages|stats|yougile. Хуки геймификации — gRPC petsvc,
# поиск/реиндекс — gRPC aisvc. env синхронизированы с dev.sh.
dev-tasks: dev-infra
	@printf "\033[1m▶ tasksvc (Go)  HTTP :8095\033[0m\n"
	cd back-go/tasks && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	PETS_GRPC_ADDR="localhost:9094" \
	AI_GRPC_ADDR="localhost:9093" \
	YOUGILE_ENC_KEY="CT5VF1jg6uFFbj4W_6RW3z3416bPlfbxdMYelrEOIXc=" \
	HTTP_ADDR=":8095" \
	go run ./cmd/tasksvc

# Realtime-шлюз: WebSocket /ws (комнаты all/user_{id}), presence в Redis,
# ринг-фаза → gRPC callsvc, доставка событий gw2:*:events клиентам.
dev-gateway: dev-infra
	@printf "\033[1m▶ gatewaysvc (Go)  HTTP :8096\033[0m\n"
	cd back-go/gateway && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	CALLS_GRPC_ADDR="localhost:9090" \
	HTTP_ADDR=":8096" \
	go run ./cmd/gatewaysvc

# Go-микросервис пуш-уведомлений: подписан на gw2:*:events, шлёт FCM
# офлайн-получателям; REST /api/push/register|unregister. Без
# FIREBASE_CREDENTIALS_JSON отправка отключена (для dev это норма).
dev-push: dev-infra
	@printf "\033[1m▶ pushsvc (Go)  HTTP :8097\033[0m\n"
	cd back-go/push && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	FIREBASE_CREDENTIALS_JSON="$${FIREBASE_CREDENTIALS_JSON:-}" \
	HTTP_ADDR=":8097" \
	go run ./cmd/pushsvc

# Go-микросервис рассылки писем: gRPC :9098 (Send), HTTP :8098 (/healthz).
# В dev письма уходят в mailpit (docker compose up поднимает его) — смотреть на
# http://localhost:8025. Реальный SMTP не нужен.
dev-mail: dev-infra
	@printf "\033[1m▶ mailsvc (Go)  gRPC :9098  HTTP :8098\033[0m\n"
	cd back-go/mail && \
	SMTP_HOST="localhost" \
	SMTP_PORT="1025" \
	SMTP_TLS="none" \
	SMTP_FROM="noreply@grovework.local" \
	HTTP_ADDR=":8098" \
	GRPC_ADDR=":9098" \
	go run ./cmd/mailsvc

# Go-микросервис реестров: REST /api/registries/*. env синхронизированы с dev.sh.
dev-registry: dev-infra
	@printf "\033[1m▶ registrysvc (Go)  HTTP :8099\033[0m\n"
	cd back-go/registry && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	UPLOAD_FOLDER="$(PWD)/uploads" \
	HTTP_ADDR=":8099" \
	go run ./cmd/registrysvc

# Go-микросервис календарей: REST /api/calendars/*. env синхронизированы с dev.sh.
dev-calendar: dev-infra
	@printf "\033[1m▶ calendarsvc (Go)  HTTP :8100\033[0m\n"
	cd back-go/calendar && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	UPLOAD_FOLDER="$(PWD)/uploads" \
	HTTP_ADDR=":8100" \
	go run ./cmd/calendarsvc

# Go-микросервис ежедневников: REST /api/diaries/*. env синхронизированы с dev.sh.
dev-diary: dev-infra
	@printf "\033[1m▶ diarysvc (Go)  HTTP :8101\033[0m\n"
	cd back-go/diary && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	HTTP_ADDR=":8101" \
	go run ./cmd/diarysvc

# Go-микросервис корпоративного портала: REST /api/portal/*. env синхронизированы с dev.sh.
dev-portal: dev-infra
	@printf "\033[1m▶ portalsvc (Go)  HTTP :8102\033[0m\n"
	cd back-go/portal && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	UPLOAD_FOLDER="$(PWD)/uploads" \
	MESSENGER_GRPC_ADDR="localhost:9092" \
	HTTP_ADDR=":8102" \
	go run ./cmd/portalsvc

# Go-микросервис заметок: REST /api/notes/*. env синхронизированы с dev.sh.
dev-notes: dev-infra
	@printf "\033[1m▶ notesvc (Go)  HTTP :8103\033[0m\n"
	cd back-go/notes && \
	DATABASE_URL="postgresql://grovework:grovework_local@localhost:5432/grovework" \
	REDIS_URL="redis://localhost:6379/0" \
	PASETO_PUBLIC_KEY="$(PASETO_PUBLIC_KEY_DEV)" \
	UPLOAD_FOLDER="$(PWD)/uploads" \
	AI_GRPC_ADDR="localhost:9093" \
	HTTP_ADDR=":8103" \
	go run ./cmd/notesvc

gen-proto:
	bash scripts/gen_proto.sh

dev-stop:
	cd deploy && docker compose --profile full stop
	@printf "\033[32m✓ Dev-контейнеры остановлены\033[0m\n"

# Полный стек в контейнерах — прод-подобная проверка сборки/композиции
# (все Go-сервисы и nginx из образов; фронт на http://localhost:8080).
dev-stack:
	@printf "\033[1m▶ Полный стек в Docker...\033[0m\n"
	cd deploy && docker compose --profile full up -d --build
	@printf "\033[32m✓ Фронт http://localhost:8080  API http://localhost:8080/api\033[0m\n"

dev-stack-stop:
	cd deploy && docker compose --profile full stop
	@printf "\033[32m✓ Полный стек остановлен\033[0m\n"

# ── Деплой ───────────────────────────────────────────────────────
.PHONY: push push-all deploy deploy-only apk deploy-apk desktop deploy-desktop release logs status restart shell

# Прод-стек = база + оверлей (см. шапку deploy/docker-compose.prod.yml).
COMPOSE_PROD := docker compose -f docker-compose.yml -f docker-compose.prod.yml
# Сервис для logs/restart/shell: make logs s=calls (по умолчанию gateway).
s ?= gateway

# Сборка прод-образов (linux/amd64) и push в Docker Hub
# osipovskijdima/groove_work (теги migrate/gateway/calls/auth/messenger/ai/
# pets/tasks/front + версионные). Нужен одноразовый docker login.
# По умолчанию (и в `make deploy`) пушит ТОЛЬКО изменившиеся образы
# (git diff origin/main..рабочее дерево; back-go/pkg/* → все Go-сервисы).
# Выборочно:    make push only="gateway front"
# Принудительно всё (игнорируя git-дифф): make push-all
push:
	bash scripts/build_push.sh $(if $(strip $(only)),$(only),--changed)

# Принудительная пересборка и push ВСЕХ образов, без оглядки на git-дифф.
# Без аргументов build_push.sh берёт весь ALL_SERVICES (единый список в скрипте).
push-all:
	bash scripts/build_push.sh

deploy: push deploy-only

# Деплой БЕЗ сборки/пуша образов: когда make push уже сделан и осталось
# только довезти код и перезапустить стек на сервере.
deploy-only:
	@printf "\033[1m▶ Пушу в GitHub...\033[0m\n"
	git push
	@printf "\033[1m▶ Деплою на $(SERVER_HOST)...\033[0m\n"
	@# Сервер НЕ собирает образы. Вся серверная логика — в
	@# scripts/deploy_server.sh (приезжает тем же git reset): синк .env с
	@# автогенерацией недостающих секретов, firewall, compose pull +
	@# up --no-build --remove-orphans, reload nginx, health-чеки.
	$(SSH) "cd $(SERVER_DIR) && git fetch origin && git reset --hard origin/main && bash scripts/deploy_server.sh"
	@printf "\033[32m✓ Задеплоено на $(SERVER_HOST)\033[0m\n"

# Сборка мобильного приложения (Capacitor-обёртка mobile/, тонкая — UI приезжает
# с прод-сервера): подписанный release-APK → apps/mobile/groovework.apk (канал
# приложения, зеркало apps/desktop/) + КОПИЯ в apps/groovework.apk со
# синхронизацией apps/version.json (старый канал: установленные ранее нативные
# приложения проверяют обновления по нему и сразу получают эту же сборку).
# Подпись — mobile/android/keystore.properties (ТОТ ЖЕ ключ
# groovework-release.jks, что у прежнего нативного приложения — иначе APK не
# встанет поверх установленных). Номер сборки (versionCode, ГГММДДН) Gradle
# читает из apps/mobile/version.json — обнови current_build перед релизом.
ANDROID_DIR := mobile/android
apk:
	@if [ ! -f $(ANDROID_DIR)/keystore.properties ]; then \
		printf "\033[31m✗ Нет $(ANDROID_DIR)/keystore.properties — скопируй ключ и заполни (см. keystore.properties.example)\033[0m\n"; exit 2; fi
	@if [ ! -d mobile/node_modules ]; then cd mobile && npm install; fi
	cd mobile && npx cap sync android
	@printf "\033[1m▶ Собираю подписанный release-APK (Capacitor)...\033[0m\n"
	cd $(ANDROID_DIR) && ./gradlew assembleRelease
	@mkdir -p apps/mobile
	cp $(ANDROID_DIR)/app/build/outputs/apk/release/app-release.apk apps/mobile/groovework.apk
	cp apps/mobile/groovework.apk apps/groovework.apk
	cp apps/mobile/version.json apps/version.json
	@bash scripts/check_apk_version.sh apps/mobile/groovework.apk apps/mobile/version.json
	@printf "\033[32m✓ Готово: apps/mobile/groovework.apk (+ зеркало apps/) — выложить: make deploy-apk\033[0m\n"

# Публикация мобильного приложения: заливаем оба канала из локального apps/ в
# apps/ репозитория на сервере (его монтирует nginx в /apps/): apps/mobile/
# (сайт и автообновление обёртки смотрят сюда) и зеркало apps/groovework.apk +
# apps/version.json (по нему обновляются установленные ранее нативные
# приложения — сразу до текущей сборки обёртки). version.json хранятся и в git
# (их читает Gradle как versionCode), но scp обновляет сборку на сервере сразу.
deploy-apk:
	@if [ ! -f apps/mobile/groovework.apk ] || [ ! -f apps/groovework.apk ]; then \
		printf "\033[31m✗ Нет APK — сначала make apk\033[0m\n"; exit 2; fi
	@bash scripts/check_apk_version.sh apps/mobile/groovework.apk apps/mobile/version.json
	@bash scripts/check_apk_version.sh apps/groovework.apk apps/version.json
	@printf "\033[1m▶ Заливаю мобильное приложение на $(SERVER_HOST)...\033[0m\n"
	$(SSH) "mkdir -p $(SERVER_DIR)/apps/mobile"
	scp -i $(SSH_KEY) apps/mobile/groovework.apk apps/mobile/version.json $(SERVER_USER)@$(SERVER_HOST):$(SERVER_DIR)/apps/mobile/
	scp -i $(SSH_KEY) apps/groovework.apk apps/version.json $(SERVER_USER)@$(SERVER_HOST):$(SERVER_DIR)/apps/
	@printf "\033[32m✓ APK и version.json выложены (оба канала) — проверка обновлений увидит новую сборку\033[0m\n"

# Сборка десктоп-клиента (Electron, desktop/): dmg + NSIS exe + AppImage.
# Артефакты и version.json — в apps/desktop/ (готовы к make deploy-desktop).
# Новая версия: make desktop V=1.0.5 — пишется в desktop/package.json
# (versionCode обёртки; сам UI приезжает с сервера).
# ИМЕНА АРТЕФАКТОВ СОДЕРЖАТ ВЕРСИЮ (artifactName в desktop/package.json):
# GrooveWork-<v>-mac.dmg / -win.exe / -linux.AppImage — безымянные версии
# «GrooveWork-mac.dmg» однажды привели к раздаче старого установщика под
# новым version.json (кэш + не перезаписанный файл). version.json несёт
# карту files — обёртка и карточка скачивания читают имена оттуда.
# Сборки НЕ подписаны: для подписи нужны Apple Developer ID / win-сертификат
# (env CSC_LINK/CSC_KEY_PASSWORD electron-builder'а).
desktop:
	cd desktop && npm install
	@if [ -n "$(V)" ]; then cd desktop && npm version $(V) --no-git-tag-version --allow-same-version; fi
	@printf "\033[1m▶ Собираю десктоп-клиент (mac + win + linux)...\033[0m\n"
	cd desktop && npx electron-builder -mwl
	@mkdir -p apps/desktop
	@node -e "const fs=require('fs'); \
		const v=require('./desktop/package.json').version; \
		const files={mac:\`GrooveWork-\$${v}-mac.dmg\`, win:\`GrooveWork-\$${v}-win.exe\`, linux:\`GrooveWork-\$${v}-linux.AppImage\`}; \
		for (const f of fs.readdirSync('apps/desktop')) \
			if (/^GrooveWork-.*\.(dmg|exe|AppImage)$$/.test(f)) fs.rmSync('apps/desktop/'+f); \
		for (const name of Object.values(files)) { \
			if (!fs.existsSync('desktop/dist/'+name)) { console.error('✗ Нет desktop/dist/'+name+' — сборка не дала артефакт'); process.exit(2); } \
			fs.copyFileSync('desktop/dist/'+name, 'apps/desktop/'+name); } \
		fs.writeFileSync('apps/desktop/version.json', JSON.stringify({current_version:v, files}, null, 2)+'\n'); \
		console.log('✓ apps/desktop готов (v'+v+') — выложить: make deploy-desktop')"

# Публикация десктоп-клиента: артефакты + version.json → apps/desktop/ на
# сервере (nginx раздаёт /apps/; оттуда обёртка проверяет обновления, а
# карточка «О приложении» даёт скачать). Имена файлов — из version.json.
deploy-desktop:
	@node -e "const m=require('./apps/desktop/version.json'); \
		for (const f of Object.values(m.files)) \
			if (!require('fs').existsSync('apps/desktop/'+f)) { console.error('✗ Нет apps/desktop/'+f+' — сначала make desktop'); process.exit(2); }"
	@printf "\033[1m▶ Заливаю десктоп-клиент на $(SERVER_HOST)...\033[0m\n"
	$(SSH) "mkdir -p $(SERVER_DIR)/apps/desktop"
	cd apps/desktop && scp -i $(SSH_KEY) $$(node -p "Object.values(require('./version.json').files).join(' ')") version.json $(SERVER_USER)@$(SERVER_HOST):$(SERVER_DIR)/apps/desktop/
	@printf "\033[32m✓ Десктоп-клиент выложен — карточка в «О приложении» и автопроверка обновлений увидят новую версию\033[0m\n"

# ── Полный релиз одной командой ──────────────────────────────────────
# make release MSG="текст коммита" [V=1.0.6] [BUILD=2607111]
#   1) коммитит рабочее дерево (MSG обязателен, если есть изменения);
#   2) поднимает версии приложений: BUILD → apps/mobile/version.json
#      (без BUILD versionCode инкрементируется сам по схеме ГГММДДН),
#      V → desktop/package.json (без V версия обёртки не меняется);
#   3) собирает APK и десктоп-установщики;
#   4) докоммичивает обновлённые version.json;
#   5) make deploy (образы → Docker Hub, git push, выкат сервера) и
#      заливает артефакты обоих приложений (deploy-apk + deploy-desktop).
release:
	@if [ -n "$$(git status --porcelain)" ] && [ -z "$(MSG)" ]; then \
		printf "\033[31m✗ Есть незакоммиченные изменения — передай MSG=\"текст коммита\"\033[0m\n"; exit 2; fi
	@if [ -n "$(MSG)" ] && [ -n "$$(git status --porcelain)" ]; then \
		git add -A && git commit -m "$(MSG)"; fi
	@node -e "const fs=require('fs'); const p='apps/mobile/version.json'; \
		const m=JSON.parse(fs.readFileSync(p)); \
		const today=new Date().toLocaleDateString('sv',{timeZone:'Europe/Moscow'}).slice(2).replaceAll('-',''); \
		const next=$(if $(BUILD),$(BUILD),String(m.current_build).startsWith(today) ? m.current_build+1 : Number(today+'0')); \
		m.current_build=next; fs.writeFileSync(p, JSON.stringify(m, null, 2)+'\n'); \
		console.log('▶ Мобильная сборка: '+next)"
	$(MAKE) apk
	$(MAKE) desktop $(if $(V),V=$(V),)
	@if [ -n "$$(git status --porcelain apps desktop/package.json)" ]; then \
		git add apps desktop/package.json && \
		git commit -m "Версии приложений: десктоп $$(node -p "require('./desktop/package.json').version"), мобильная сборка $$(node -p "require('./apps/mobile/version.json').current_build")"; fi
	$(MAKE) deploy
	$(MAKE) deploy-apk
	$(MAKE) deploy-desktop
	@printf "\033[32m✓ Релиз выкачен целиком: сервер, десктоп, мобильное приложение\033[0m\n"

logs:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) logs -f --tail=200 $(s)"

status:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) ps"

restart:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) restart $(s)"
	@printf "\033[32m✓ $(s) перезапущен\033[0m\n"

shell:
	$(SSH) "cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) exec $(s) sh -c 'command -v bash >/dev/null && exec bash || exec sh'"

# ── Бэкап БД ─────────────────────────────────────────────────────
# make backup — pg_dump прод-БД (внутри контейнера db, креды берутся из
# его же POSTGRES_*) → gzip на сервере → стрим по SSH в backups/ (в
# .gitignore: дамп содержит реальные данные). Флаги --clean --if-exists
# --no-owner — чтобы дамп накатывался на локальную dev-БД одной командой:
#   gunzip -c backups/gw2_<дата>.sql.gz | docker exec -i deploy-db-1 psql -U grovework -d grovework
.PHONY: backup
BACKUP_TS := $(shell date +%Y%m%d-%H%M%S)
BACKUP_FILE := backups/gw2_$(BACKUP_TS).sql.gz
backup:
	@mkdir -p backups
	@printf "\033[1m▶ Дамп прод-БД с $(SERVER_HOST)...\033[0m\n"
	@$(SSH) "set -o pipefail; cd $(SERVER_DIR)/deploy && $(COMPOSE_PROD) exec -T db sh -c 'pg_dump --clean --if-exists --no-owner -U \$$POSTGRES_USER -d \$$POSTGRES_DB' | gzip -c" > $(BACKUP_FILE).part \
		|| { rm -f $(BACKUP_FILE).part; printf "\033[31m✗ Дамп не удался\033[0m\n"; exit 1; }
	@gunzip -t $(BACKUP_FILE).part && mv $(BACKUP_FILE).part $(BACKUP_FILE)
	@printf "\033[32m✓ Дамп: $(BACKUP_FILE) ($$(du -h $(BACKUP_FILE) | cut -f1))\033[0m\n"

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

# ── Демо-данные в локальной dev-БД ───────────────────────────────
# Компания «Грув Демо» с сотрудниками, грувиками во всех состояниях
# (голодный/простуженный/грязнуля/хандра перед побегом/одинокий/в пути),
# порталом с ветками комментариев и лайками, задачами и юнитами.
# Идемпотентно: прежний посев чистится. Пароль аккаунтов: demo1234
.PHONY: dev-seed
dev-seed:
	@printf "\033[1m▶ Заливаю демо-данные в dev-БД...\033[0m\n"
	@bash scripts/seed_dev.sh

# ── Сброс пароля суперадмина в локальной dev-БД ──────────────────
# Использование: make dev-reset NEWPASS='новый-пароль'
# То же, что make reset, но без SSH — работает с контейнером db dev-инфры.
.PHONY: dev-reset
dev-reset:
	@if [ -z "$(NEWPASS)" ]; then \
		printf "\033[31m✗ Передайте новый пароль:\033[0m  make dev-reset NEWPASS='новый-пароль'\n"; \
		exit 2; \
	fi
	@printf "\033[1m▶ Сбрасываю пароль суперадмина в dev-БД...\033[0m\n"
	@scripts/reset_superadmin_password_dev.sh "$(NEWPASS)"
