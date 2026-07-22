-- +goose Up
-- Личные папки чатов (по образцу Telegram). Папка принадлежит одному
-- пользователю, кросс-компанийна (как сам мессенджер) и синхронизируется между
-- устройствами. Эффективный состав папки = ручные привязки (chat_folder_items)
-- ∪ чаты, подходящие под включённые авто-фильтры (include_personal/groups/unread) —
-- фильтры вычисляются на клиенте, здесь хранятся только флаги.
CREATE TABLE chat_folders (
    id               BIGSERIAL PRIMARY KEY,
    owner_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title            TEXT   NOT NULL,
    emoji            TEXT,
    position         INT    NOT NULL DEFAULT 0,
    include_personal BOOLEAN NOT NULL DEFAULT FALSE,
    include_groups   BOOLEAN NOT NULL DEFAULT FALSE,
    include_unread   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_chat_folders_owner ON chat_folders(owner_id, position);

-- Ручная привязка чата к папке (many-to-many). Составной PK — строки без
-- собственного id (бэкап-импорт setval'ит только таблицы с колонкой id).
CREATE TABLE chat_folder_items (
    folder_id       BIGINT NOT NULL REFERENCES chat_folders(id) ON DELETE CASCADE,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    PRIMARY KEY (folder_id, conversation_id)
);
CREATE INDEX idx_chat_folder_items_conv ON chat_folder_items(conversation_id);

-- +goose Down
DROP TABLE IF EXISTS chat_folder_items;
DROP TABLE IF EXISTS chat_folders;
