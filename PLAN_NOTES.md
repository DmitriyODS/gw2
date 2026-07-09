# План реализации раздела «Заметки» (notesvc + фронт)

Самодостаточное задание для агента. Перед началом прочитать `CLAUDE.md`
(архитектура и правила) и `front/DESIGN.md` (стиль «Expressive Glass» и готовые
паттерны). Правила проекта обязательны: межсервисное общение — только gRPC
(здесь межсервисных вызовов нет), никаких hex-цветов в компонентах — только
токены, комментарии по-русски и только по делу, `data/changelog.json` НЕ трогать,
ничего не коммитить без команды пользователя.

## 1. Что уже есть (не переделывать)

- Хаб «Заметки»: пункт в сайдбаре/нижней навигации ведёт на `/notes`;
  `front/src/components/notes/NotesHubTabs.vue` — вкладки «Заметки» (`/notes`,
  сейчас заготовка `NotesView.vue`) и «Ежедневник» (`/diaries`, рабочий).
- Стиль: glass-каркас мастер-детейл готов — глобальные классы `.split-view`,
  `.split-side`, `.split-side-head/-tile/-title`, `.split-side-list`,
  `.split-side-item` (+`.split-item-tile`, `.split-side-name`), `.split-side-add`,
  `.split-main`, `.split-empty` в `front/src/assets/main.css`; кнопки `.btn-grad`,
  `.btn-glass`; чипы `.chip-tint--*`; поле поиска `components/common/SearchField.vue`;
  `EmptyState` с `tone="soft"`. Эталоны: `RegistriesView.vue`, `DiaryView.vue`.
- Ежедневник (`diarysvc`, :8101) — образец личного кросс-компанийного сервиса:
  скоуп по владельцу, шаринг код-capability, роут без требования компании.

## 2. Продуктовые требования (со слов пользователя, выполнять точно)

1. Вкладка «Заметки» — две колонки. Слева — группы заметок: первая всегда
   «Все» (все заметки), ниже пользовательские группы, кнопка «Добавить группу».
2. Заметка может принадлежать нескольким группам, одной или ни одной; выбор
   групп — в настройках конкретной заметки.
3. Справа — заметки-«стикеры» плитками: заголовок + дата и время создания.
4. Клик по плитке открывает СТРАНИЦУ заметки (не модалку): редактируются
   название и тело.
5. Тело — rich-редактор с live-форматированием (выделил слово → нажал
   «жирный» → стало жирным) и панелью форматирования сверху: заголовки/
   подзаголовки, жирный, курсив, подчёркивание, зачёркивание, выделение
   цветом, ссылки, код, изображения, таблицы, списки и т.п.
6. На заметку создаются ссылки, которые можно отправить кому угодно; при
   создании выбирается режим: «только чтение» или «чтение и редактирование».
7. Экспорт заметки в .txt и импорт заметки из .txt.
8. Заметки строго приватные для каждого пользователя (кросс-компанийные,
   как ежедневник) — пользователи друг друга не видят.

## 3. Бэкенд: новый микросервис notesvc (`back-go/notes`)

Шаблон — `back-go/diary` (ближайший по скоупу), плюс `pkg/storage` из
registry/calendar (нужны картинки редактора). Слои стандартные:
`internal/domain → service → repository/postgres → transport/http` +
`internal/endpoint`. HTTP **:8103**, gRPC НЕТ. События → Redis-канал
`gw2:notes:events` (через `pkg/events`). Авторизация локальная
(`pkg/pasetoauth.RequireToken` — активная компания НЕ нужна, как в diarysvc);
скоуп ВСЕХ запросов — `owner_id` из токена.

### Схема БД (goose-миграция в `back-go/migrate/migrations`, следующий номер после существующих — проверить `ls`)

- `notes`: id BIGSERIAL PK, owner_id INT NOT NULL REFERENCES users ON DELETE CASCADE,
  title VARCHAR(300) NOT NULL DEFAULT '', doc JSONB NOT NULL DEFAULT '{}'
  (документ TipTap), text_content TEXT NOT NULL DEFAULT '' (плоский текст для
  поиска и txt-экспорта, пересчитывается сервером из doc при сохранении),
  created_at/updated_at TIMESTAMPTZ. Индексы: owner_id; триграммный GIN по
  (title || ' ' || text_content) — `pg_trgm` уже включён в БД.
- `note_groups`: id, owner_id (FK users CASCADE), name VARCHAR(100), position INT,
  created_at. Индекс owner_id.
- `note_group_items`: note_id FK notes CASCADE, group_id FK note_groups CASCADE,
  PRIMARY KEY (note_id, group_id).
- `note_shares`: id, note_id FK CASCADE, code VARCHAR UNIQUE
  (генерация — `pkg/records.NewShareCode`), access VARCHAR(8) NOT NULL
  CHECK (access IN ('view','edit')), created_at.

### REST `/api/notes` (формат ошибок `{"error": CODE, "message": ...}`)

- `GET /api/notes?group_id=&search=` — список плиток владельца
  (id, title, excerpt из text_content, created_at, updated_at, group_ids);
  сортировка updated_at DESC. `GET /api/notes/:id` — полная (с doc).
- `POST /api/notes` (пустая или `{title}`), `PATCH /api/notes/:id`
  (`{title?, doc?}` — сервер пересчитывает text_content), `DELETE /api/notes/:id`
  (чистит файлы заметки в storage).
- Группы: `GET/POST /api/notes/groups`, `PATCH/DELETE /api/notes/groups/:id`
  (удаление группы НЕ удаляет заметки — только связи).
- `PUT /api/notes/:id/groups` — `{group_ids: []}` (все свои).
- Шаринг: `GET/POST /api/notes/:id/shares` (`{access}`),
  `DELETE /api/notes/:id/shares/:shareId`.
- Публичные (БЕЗ авторизации; в Fiber мидлварь группы на префиксе — пропустить
  `/api/notes/shared/*`, как в registrysvc): `GET /api/notes/shared/:code` —
  заметка + access; `PUT /api/notes/shared/:code` — тело `{title?, doc?}`,
  ТОЛЬКО если access='edit'. Троттлинг записи по коду (Redis, напр. 30 req/мин,
  fail-open) — от вандализма.
- Файлы: `POST /api/notes/:id/uploads` (картинки редактора, только владелец) —
  через `pkg/storage` `FileStore("notes")`, лимит как у мессенджера; отдача —
  штатный `/uploads/<key>`. Удаление заметки чистит её файлы.
- Экспорт: `GET /api/notes/:id/export` → text/plain attachment
  (`title + "\n\n" + text_content`). Импорт: `POST /api/notes/import`
  (multipart .txt; первая строка → title, остальное → doc из параграфов).
- Сокет-события в комнату `user_{owner_id}`: `note:created/updated/deleted`,
  `note_group:created/updated/deleted` (нейминг с префиксом note_ — не
  пересекаться с событиями других сервисов).

### Тесты notesvc

`go test ./...` на фейках портов: скоуп по владельцу (чужая заметка → 404),
edit-share пишет / view-share не пишет, пересчёт text_content, экспорт/импорт,
удаление группы не трогает заметки.

## 4. Инфраструктура (по образцу diarysvc/registrysvc — пройтись по КАЖДОМУ пункту)

- `deploy/docker-compose.yml` (+`override`, `prod`): сервис notes, healthcheck,
  в цепочке depends_on после migrate; env: DB/Redis, PASETO_PUBLIC_KEY,
  STORAGE_* (uploads-том как у registry).
- Маршрутизация: nginx (`deploy/nginx/nginx.conf` и `nginx.prod.conf`) —
  `/api/notes` → notes:8103 (длинные префиксы раньше `/api/`); vite-proxy в
  `front/vite.config.js`. ПОМНИ: правки nginx на сервере требуют
  `up -d --force-recreate --no-deps nginx` (bind-mount inode).
- gateway: убедиться, что канал `gw2:notes:events` доставляется (проверить,
  как gatewaysvc подписывается — список каналов или psubscribe; если список —
  добавить канал И в `back-go/apitest` харнесс: там уже ловили пропущенные
  каналы calendar/diary).
- `Makefile` (`dev-notes`), `dev.sh`, `scripts/build_push.sh` (тег `notes`),
  `scripts/deploy_server.sh` (healthz-чек), `back-go/go.work`,
  `back-go/notes/Dockerfile` (копия diary-Dockerfile).
- Бэкап: новые таблицы попадают в универсальный дамп автоматически; добавить
  раздел «Заметки» в `domain.BackupSections` (authsvc) и зеркально в
  `front/src/utils/backupSections.js`.
- apitest: поднять notesvc (порт +10000), сквозной сценарий: создание → шаринг
  edit → анонимная правка по коду → видна владельцу.

## 5. Фронтенд

### Редактор — TipTap (@tiptap/vue-3)

Пакеты: `@tiptap/vue-3`, `@tiptap/pm`, `@tiptap/starter-kit` + расширения:
`underline`, `link`, `image`, `table` (+row/cell/header), `highlight`
(multicolor), `task-list`/`task-item`, `placeholder`, `character-count` (опц.).
ВАЖНО (память проекта): `package-lock.json` регенерировать ТОЛЬКО в
linux-контейнере `node:24-alpine` — macOS-npm теряет optional-deps, и
`npm ci` на сервере падает (рецепт в memory/`project_npm_lockfile_linux.md`).

Хранение — TipTap JSON (`doc`), не markdown: highlight-цвета и таблицы в чистом
md не выражаются. Палитра highlight — из токенов задач `--tag-*-surface`.

### Структура

- `front/src/api/notes.js` (вручную), `front/src/stores/notes.js`
  (notes list + groups + активная группа; события идемпотентны),
  `front/src/socket/notes.js` (+регистрация в `socket/index.js`).
- `NotesView.vue` (заменить заготовку) — `.split-view`:
  - `.split-side`: шапка — `NotesHubTabs` (ОСТАВИТЬ — это переключатель хаба);
    ниже пункт «Все» (иконка apps/notes, счётчик заметок `.rail-badge`-стилем),
    пользовательские группы `.split-side-item` (иконка folder, счётчик),
    внизу `.split-side-add` «Добавить группу» (инлайн-инпут или AppDialog).
    Контекст-действия группы: переименовать/удалить (мини-меню или иконки по hover).
  - `.split-main`: тулбар — `SearchField` (поиск по title+тексту, дебаунс 300мс,
    серверный `?search=`) + `.btn-grad` «+ Заметка» (создаёт пустую и сразу
    открывает страницу). Сетка плиток-«стикеров»: `--acrylic-card-bg` +
    `--acrylic-border`, radius ~18px, заголовок (2 строки clamp), excerpt dim
    (2-3 строки), низ — дата и время создания (`calendar_today`, dd.mm.yyyy HH:mm).
    Пустые состояния: нет заметок вообще / пусто в группе / ничего не найдено
    (`EmptyState tone="soft"`).
- Роут `/notes/:id(\\d+)` → `NoteEditorView.vue` (`meta.requiresAuth`, БЕЗ
  requiresCompany). Страница: кнопка «← Заметки», инпут заголовка (крупный,
  прозрачный), sticky-панель форматирования (`--acrylic-bg-strong` + blur —
  правило «sticky-шапки» из DESIGN.md): H1/H2/H3, B/I/U/S, highlight-палитра,
  ссылка (промпт/поповер), код-инлайн и код-блок, картинка (upload → вставка
  `/uploads/<key>`), таблица, списки (маркированный/нумерованный/чекбоксы),
  undo/redo. Справа в шапке: «Группы» (диалог с чипами-мультиселектом),
  «Поделиться» (диалог по образцу `DiaryShareDialog`: создать ссылку
  чтение/редактирование, список ссылок, копирование, отзыв), «Экспорт .txt»,
  «Удалить» (ConfirmDialog). Автосохранение: дебаунс 1.5с после правок +
  немедленно на blur/beforeunload/Cmd+S; индикатор «Сохранено/Сохраняю…».
- Импорт .txt: кнопка в тулбаре списка (`.btn-glass`, input type=file accept=.txt).
- `SharedNoteView.vue` — публичный роут `/note/:code` (`meta.public`; НЕ
  конфликтует с `/notes`): view — рендер только для чтения (`editable: false`);
  edit — тот же редактор, сохранение PUT по коду, без панели «Поделиться/Удалить».
- Мобайл (≤768px): по образцу DiaryView — левая панель скрыта, лента групп
  чипами над плитками (как `dv-regstrip`), плитки в 1 колонку, редактор
  полноэкранный, панель форматирования горизонтально скроллится.
- Активность пункта «Заметки» в сайдбаре уже покрывает `/notes/...`
  (startsWith) — не трогать.

### Тесты фронта

vitest: стор (идемпотентность сокет-событий, фильтр по группе), утилита
doc→plain text если делается на фронте (лучше на бэке — тогда не нужна).

## 6. Документация

- `CLAUDE.md`: добавить notesvc везде, где перечислены сервисы (таблица стека,
  «Структура директорий», «Архитектура» — абзац сервиса, «Маршрутизация»,
  dev-команды, деплой-теги, тесты). Кратко, в стиле соседних абзацев.
- `front/DESIGN.md`: при появлении новых переиспользуемых паттернов (панель
  форматирования, плитка-стикер) — дополнить.
- `data/changelog.json` — НЕ ТРОГАТЬ.

## 7. Порядок работ и критерии готовности

1. Миграция + notesvc (domain/service/repo/http) + unit-тесты → `go test ./...` зелёный.
2. Инфраструктура (compose, nginx, vite, Makefile/dev.sh, gateway-канал) →
   `./dev.sh`-стек поднимается, `/api/notes` отвечает.
3. Фронт: стор/апи/сокеты → список+группы → страница-редактор → шаринг →
   экспорт/импорт → мобайл. `npm run build` и `npx vitest run` зелёные.
4. apitest-сценарий. CLAUDE.md. Финальная проверка `make dev-stack` не обязательна.

Критерии: все 8 продуктовых пунктов из §2 работают; стиль — только токены и
готовые glass-паттерны; ежедневник и хаб-вкладки не сломаны; тесты зелёные.
