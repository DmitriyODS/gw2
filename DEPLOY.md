# Groove Work v2.0 — Деплой на сервер

## Требования

- Docker 24+ и Docker Compose v2
- Открытый порт 80 (или 443 при HTTPS)
- Для звонков (LiveKit): открытые порты **7881/TCP** и **7882/UDP** (медиа) + **5349/TCP** и **3478/UDP** (встроенный TURN-relay для NAT/VPN/мобильных сетей). `deploy_server.sh` открывает их сам при активном ufw; вручную: `ufw allow 7881/tcp && ufw allow 7882/udp && ufw allow 5349/tcp && ufw allow 3478/udp`. TURN/TLS использует тот же сертификат Let's Encrypt, что и nginx (`LIVEKIT_TURN_DOMAIN`, по умолчанию `gw.kodass.ru`).
- Доступ к Docker Hub: сервер **не собирает** образы приложений, а тянет
  готовые из репозитория `osipovskijdima/groove_work` (теги
  `app` / `calls` / `auth` / `front`). Пушит их локальная машина:
  `make push` (= `scripts/build_push.sh`, сборка под `linux/amd64`).
  Если репозиторий приватный — один раз выполнить `docker login` и на
  локальной машине (push), и на сервере (pull).

---

## Первый запуск

### 1. Настроить переменные окружения

```bash
cp deploy/.env.example deploy/.env
```

> **Можно пропустить:** `make deploy` сам запускает на сервере
> `scripts/deploy_server.sh`, который создаёт `deploy/.env` из примера и
> **генерирует все недостающие секреты автоматически** (существующие значения
> никогда не перезаписывает, перед правкой делает бэкап `.env.bak.<дата>`).
> Вручную заполнить остаётся только `YOUGILE_WEBHOOK_PUBLIC_BASE` (публичный
> URL — его не угадать). Проверить/подготовить .env без выката:
> `bash scripts/deploy_server.sh --env-only`.

Открыть `deploy/.env` и выставить **реальные** значения:

| Переменная | Что поставить |
|---|---|
| `DB_PASSWORD` | Сильный пароль (минимум 20 символов) |
| `PASETO_PRIVATE_KEY` / `PASETO_PUBLIC_KEY` | Пара Ed25519 (hex) для access-токенов PASETO — генерируются ВМЕСТЕ, deploy-скрипт сделает сам |
| `PASETO_REFRESH_KEY` | Случайные 32 байта hex — ключ refresh-токенов (v4.local) |
| `SECRET_KEY` | Случайная строка ≥ 32 символа |
| `AI_KEY_ENCRYPTION_KEY` | Fernet-ключ для шифрования AI-ключей компаний |
| `YOUGILE_ENC_KEY` | Fernet-ключ для шифрования персональных YouGile-ключей пользователей |
| `YOUGILE_WEBHOOK_PUBLIC_BASE` | Публичный URL приложения (например `https://gw.example.com`) — нужен для регистрации webhook'а YouGile. Без него двусторонняя синхра не подключится, но импорт/экспорт работают. |
| `LIVEKIT_API_KEY` | Идентификатор ключа LiveKit — **обязателен**, без него compose не поднимется (deploy-скрипт сгенерирует сам) |
| `LIVEKIT_API_SECRET` | Случайная строка ≥ 32 символа — подпись токенов звонков и вебхуков LiveKit (deploy-скрипт сгенерирует сам) |

Сгенерировать случайные секреты:
```bash
# SECRET_KEY / PASETO_REFRESH_KEY:
python3 -c "import secrets; print(secrets.token_hex(32))"
# PASETO_PRIVATE_KEY/PASETO_PUBLIC_KEY — пара Ed25519, рецепт в deploy/.env.example
# (или просто доверьте deploy_server.sh — он генерирует пару сам)
# AI_KEY_ENCRYPTION_KEY / YOUGILE_ENC_KEY:
python3 -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())"
```

### YouGile-интеграция: сетевые требования

| Что нужно открыть | Зачем |
|---|---|
| **Egress** на `https://ru.yougile.com:443` из tasks-контейнера | Все исходящие вызовы YouGile API (auth, проекты, задачи, чат карточек, регистрация webhook). |
| **Ingress** на `<YOUGILE_WEBHOOK_PUBLIC_BASE>/api/yougile/webhook/*` снаружи (TLS!) | YouGile стучится сюда событиями `task-*`. Nginx проксирует `/api/yougile` на tasks:8095 (вебхук в tasksvc публичный, без токена). |

Что НЕ нужно: открывать для YG медиа-порты LiveKit (7881/7882 — это для звонков), отдельный route в nginx, отдельный subdomain.

Авторизация ingress'а — через `secret`, который мы сами генерируем при включении интеграции и подставляем в URL webhook'а (`/webhook/<companyId>/<secret>`). Утечка `secret` ≡ возможность писать в чужие задачи; ключи компании это не вскрывает.

### Файловое хранилище: S3 Beget (prod)

Весь пользовательский контент (вложения мессенджера, картинки/файлы реестров и
календарей, аватарки) в проде хранится в S3-бакете Beget (`pkg/storage`,
`STORAGE_BACKEND=s3`). Один раз настроить:

1. **Создать бакет** в панели Beget (Облако → Хранилище S3), получить
   `Access Key` и `Secret Key`.
2. **Прописать в `deploy/.env`:**
   ```
   STORAGE_BACKEND=s3
   S3_ENDPOINT=s3.ru1.storage.beget.cloud
   S3_REGION=ru1
   S3_BUCKET=<имя-бакета>
   S3_ACCESS_KEY=<ключ>
   S3_SECRET_KEY=<секрет>
   ```
3. **Имя бакета в `deploy/nginx/nginx.prod.conf`** (`location /uploads/` →
   `proxy_pass https://s3.ru1.storage.beget.cloud/<имя-бакета>/;`) — закоммитить.
   После правки nginx-конфига нужен `--force-recreate` контейнера nginx.
4. **Перенести существующие файлы** (один раз; идемпотентно — можно повторять):
   ```
   bash scripts/migrate_uploads_s3.sh
   ```
   Скрипт гоняет `pkg/cmd/uploadmigrate` в одноразовом golang-контейнере: читает
   том `deploy_uploads` и заливает объекты в бакет с `public-read` ACL.

**Про доступ:** ключ хранилища Beget НЕ имеет прав на политику бакета, поэтому
публичное чтение обеспечивается **ACL на уровне каждого объекта** — сервисы
ставят `public-read` при загрузке (`pkg/storage`), миграция — тоже. Делать весь
бакет публичным не требуется (и Beget это не рекомендует). Файлы доступны по
неугадываемому пути через nginx — прежняя модель доступа сохранена.

Контракт `/uploads/<key>` не меняется: nginx проксирует его на бакет, фронт и
Android ничего не знают о бэкенде. В dev `STORAGE_BACKEND` не задают — файлы
лежат на диске (`uploads/`), раздаёт vite/nginx.

### 2. Запустить

С локальной машины (предпочтительно):

```bash
make deploy   # = make push (сборка+публикация образов) → git push →
              #   на сервере: git reset + scripts/deploy_server.sh
```

Вручную на сервере (образы уже должны быть запушены через `make push`):

```bash
cd deploy
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --no-build
```

> **Важно про compose-файлы.** Прод-стек собирается из пары
> `docker-compose.yml` (база) + `docker-compose.prod.yml` (оверлей: образы
> из Docker Hub, TLS, certbot, обязательные секреты). Голый `docker compose
> up` запускать на сервере **нельзя** — он автоматически подхватит
> dev-оверлей `docker-compose.override.yml` и опубликует порты БД/Redis
> наружу. Проще всего вообще не звать compose руками: `make deploy` с
> локальной машины делает всё сам через `scripts/deploy_server.sh`.

При первом запуске:
- Сервер стянет готовые образы из Docker Hub (Go-микросервисы `migrate`,
  `gateway`, `calls`, `auth`, `messenger`, `ai`, `groove`, `tasks` и
  `front` — nginx со собранной SPA)
- Применит миграции базы данных (run-once контейнер `migrate`, goose)
- Создаст первого пользователя: **логин** `admin` / **пароль** `admin`

Приложение будет доступно по адресу: `http://<IP сервера>`

> **Важно:** сразу после первого входа смените пароль `admin` через раздел **Профиль**. Аккаунт создан без принудительной смены (`is_default_pass=FALSE`), поэтому блокировки нет — доступ полный сразу.

---

## Управление

С локальной машины (предпочтительно): `make logs` / `make status` /
`make restart` / `make shell`; для микросервиса звонков — `make logs s=calls`,
`make restart s=calls`.

На самом сервере (из `deploy/`, всегда парой `-f`):

```bash
PROD="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

$PROD ps                    # статус контейнеров
$PROD logs -f gateway       # логи шлюза (calls / auth / livekit / nginx — аналогично)
$PROD restart gateway       # перезапуск
$PROD down                  # остановка (данные сохраняются)
$PROD pull && $PROD up -d --no-build   # обновить после нового `make push`
```

---

## Обновление приложения

`make deploy` с локальной машины (`make push` образов → git push →
git reset на сервере → `scripts/deploy_server.sh`). Вручную:

```bash
# Локально: собрать и запушить образы (все или выборочно)
make push                  # или: make push only="app front"

# На сервере: обновить конфиги и перекатить контейнеры
git pull
cd deploy
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --no-build
```

Миграции применяются автоматически при старте `app`.

**Откат на предыдущую версию:** каждый push дополнительно помечает образы
версионным тегом (`app-3.6.0` и т.п. — версия из `front/package.json`).
В `deploy/.env` на сервере выставить `APP_TAG=app-3.6.0` (аналогично
`CALLS_TAG` / `AUTH_TAG` / `FRONT_TAG`) и повторить `pull` + `up -d
--no-build`. Убрать переменную — вернуться на текущие теги.

---

## Резервное копирование

С локальной машины — `make backup`: pg_dump прод-БД внутри контейнера `db`,
gzip на сервере, стрим по SSH в `backups/gw2_<дата>.sql.gz` (каталог в
.gitignore). Дамп сделан с `--clean --if-exists --no-owner`, поэтому
накатывается на локальную dev-БД одной командой:

```bash
gunzip -c backups/gw2_<дата>.sql.gz | docker exec -i deploy-db-1 psql -U grovework -d grovework
```

Через UI: **Настройки → Копирование и восстановление → Создать резервную копию**.

Или вручную — данные лежат в Docker volume:
```bash
docker run --rm \
  -v grovework_pg_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/pg_data_$(date +%Y%m%d).tar.gz -C /data .
```

---

## HTTPS (Let's Encrypt)

TLS уже встроен в прод-оверлей: nginx слушает 80/443 с конфигом
`deploy/nginx/nginx.prod.conf`, сертификаты живут во внешних volume'ах
`gw1_certbot-conf` / `gw1_certbot-www`, продление крутит контейнер
`certbot` (renew каждые 12 часов). Refresh-cookie authsvc выставляется
с `Secure` всегда — дополнительных правок кода не требуется.

---

## Структура портов и сетей

```
Интернет :80/:443             Интернет :7881/tcp :7882/udp (медиа)
    ↓                                  :5349/tcp :3478/udp (TURN-relay)
nginx (фронт + реверс-прокси)            ↓ (медиа/TURN WebRTC, мимо nginx)
    ├── /            → front/dist (Vue SPA)  livekit (SFU звонков + TURN)
    ├── /api/calls/* → calls:8090 (Go REST)  ↑
    ├── /api/...     → auth/messenger/ai/    │ сигнальный WS
    │                  groove/tasks (Go REST)│
    ├── /ws          → gateway:8096 (WS)     │
    ├── /livekit     → livekit:7880 ─────────┘
    └── /uploads     → S3-бакет Beget (prod) / volume (dev) — аватарки, вложения, файлы реестров/календарей

gateway:8096 (Go, realtime-шлюз)     calls:8090/:9090 (Go, callsvc)
    ├── db:5432 (last_seen_at)           ├── db:5432 (PostgreSQL)
    ├── redis:6379 (presence + события)  ├── redis:6379 (publish событий)
    └── calls:9090 (gRPC, ринг-фаза)     └── livekit:7880 (Twirp + вебхуки ←)
```

Микросервисы наружу не публикуются: REST ходит через nginx, gRPC доступен
только внутри docker-сети. Сокет-события сервисы публикуют в Redis-каналы
`gw2:<svc>:events`, которые слушает gateway и доставляет клиентам по WS.
