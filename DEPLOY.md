# Grove Work v2.0 — Деплой на сервер

## Требования

- Docker 24+ и Docker Compose v2
- Открытый порт 80 (или 443 при HTTPS)

---

## Первый запуск

### 1. Настроить переменные окружения

```bash
cp deploy/.env.example deploy/.env
```

Открыть `deploy/.env` и выставить **реальные** значения:

| Переменная | Что поставить |
|---|---|
| `DB_PASSWORD` | Сильный пароль (минимум 20 символов) |
| `JWT_SECRET_KEY` | Случайная строка ≥ 32 символа |
| `SECRET_KEY` | Случайная строка ≥ 32 символа |

Сгенерировать случайные ключи:
```bash
python3 -c "import secrets; print(secrets.token_hex(32))"
```

### 2. Запустить

```bash
cd deploy
docker compose up -d --build
```

При первом запуске Docker автоматически:
- Соберёт бэкенд из `back/`
- Соберёт фронтенд из `front/` (Node.js → Vite build → nginx)
- Применит миграции базы данных (`flask db upgrade`)
- Создаст первого пользователя: **логин** `admin` / **пароль** `admin`

Приложение будет доступно по адресу: `http://<IP сервера>`

> **Важно:** сразу после первого входа смените пароль `admin` через раздел **Профиль**. Аккаунт создан без принудительной смены (`is_default_pass=FALSE`), поэтому блокировки нет — доступ полный сразу.

---

## Управление

```bash
# Статус контейнеров
docker compose ps

# Логи приложения
docker compose logs -f app

# Перезапуск
docker compose restart app

# Остановка (данные сохраняются)
docker compose down

# Пересобрать и перезапустить после изменений кода
docker compose up -d --build
```

---

## Обновление приложения

```bash
git pull
cd deploy
docker compose up -d --build
```

Миграции применяются автоматически при старте `app`.

---

## Резервное копирование

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

Если нужен HTTPS — добавьте certbot или поставьте nginx-proxy перед контейнером.

После настройки HTTPS не забудьте в `back/app/api/auth.py` изменить:
```python
secure=False  →  secure=True
```
и пересобрать: `docker compose up -d --build app`.

---

## Структура портов и сетей

```
Интернет :80
    ↓
nginx (фронт + реверс-прокси)
    ├── /          → front/dist (Vue SPA)
    ├── /api/*     → app:5000 (Flask REST)
    ├── /socket.io → app:5000 (WebSocket)
    └── /uploads   → volume (аватарки)
         ↓
    app:5000 (Flask + eventlet)
         ├── db:5432 (PostgreSQL)
         └── redis:6379 (Redis)
```
