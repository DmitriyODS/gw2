-- +goose Up
-- Журнал файлов хранилища: раздел «Настройки → Хранилище» показывает, чем
-- именно занято место, и даёт удалить лишнее.
--
-- Почему журнал ведёт биллинг, а не каждый сервис у себя: размеров и имён в
-- их таблицах почти нет — картинки заметок, досок, реестров и календарей лежат
-- ссылками внутри JSONB. Собирать по ним список постфактум значило бы мерить
-- каждый объект в S3 отдельным запросом. Здесь же запись появляется в момент
-- загрузки, когда размер и имя известны даром.
--
-- user_id — ВЛАДЕЛЕЦ КВОТЫ (для файла компании это её создатель), поэтому
-- разрез совпадает с billing_storage_usage: тот остаётся быстрым счётчиком,
-- этот — расшифровкой.
CREATE TABLE public.billing_storage_files (
    storage_key varchar(300) PRIMARY KEY,
    user_id     integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    service     varchar(24) NOT NULL,
    company_id  integer REFERENCES public.companies(id) ON DELETE CASCADE,
    file_name   text NOT NULL DEFAULT '',
    size_bytes  bigint NOT NULL DEFAULT 0,
    -- Где файл лежит: вид сущности и её идентификатор (переход к источнику из
    -- настроек). Пустые допустимы — файл нередко грузится раньше сущности, и
    -- ссылку проставляет ближайшая сверка с владельцем.
    ref_kind    varchar(24) NOT NULL DEFAULT '',
    ref_id      varchar(64) NOT NULL DEFAULT '',
    ref_title   text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- Оба запроса раздела: разбивка по сервисам и «самые крупные сверху».
CREATE INDEX billing_storage_files_user_idx ON public.billing_storage_files (user_id, service);
CREATE INDEX billing_storage_files_size_idx ON public.billing_storage_files (user_id, size_bytes DESC);

-- +goose Down
DROP TABLE public.billing_storage_files;
