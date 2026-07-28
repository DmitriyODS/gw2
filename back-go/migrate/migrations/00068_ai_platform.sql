-- +goose Up
-- ИИ переходит на ПЛАТФОРМЕННЫЙ ключ proxy-api: ключ и модели задаёт супер-админ
-- в разделе «Аудит платформы», а пользователю тариф выдаёт токены доступа
-- (учёт — billingsvc). Ключи компаний больше не нужны: компанийные ИИ-функции
-- (умный поиск задач, факты ТВ-режима) работают на платформенном ключе и тратят
-- токены СОЗДАТЕЛЯ компании — если он это разрешил.

-- Единственная строка платформенных настроек ИИ.
CREATE TABLE public.ai_platform_settings (
    id              integer PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled         boolean NOT NULL DEFAULT false,
    api_key_enc     bytea,                       -- NULL — ИИ на платформе выключен
    key_hint        varchar(24) NOT NULL DEFAULT '',
    base_url        text NOT NULL DEFAULT 'https://api.proxyapi.ru/openai/v1',
    model_chat      varchar(64) NOT NULL DEFAULT 'gpt-5.4-nano',
    model_embedding varchar(64) NOT NULL DEFAULT 'text-embedding-3-small',
    model_support   varchar(64) NOT NULL DEFAULT 'gpt-5.4-nano',
    updated_at      timestamptz NOT NULL DEFAULT now()
);

INSERT INTO public.ai_platform_settings (id) VALUES (1);

-- Справочник моделей: цена за 1 млн токенов (копейки) задаёт стоимость обращения
-- в ТОКЕНАХ ДОСТУПА. Базовая единица — самая дешёвая модель: 1 токен доступа =
-- 1000 её токенов, остальные модели дороже пропорционально цене.
CREATE TABLE public.ai_models (
    code            varchar(64) PRIMARY KEY,
    title           varchar(64) NOT NULL,
    kind            varchar(16) NOT NULL DEFAULT 'chat',  -- chat | embedding
    price_per_mtok  bigint NOT NULL DEFAULT 0,            -- копейки за 1 млн токенов
    selectable      boolean NOT NULL DEFAULT true,        -- показывать пользователю в выборе
    is_active       boolean NOT NULL DEFAULT true,
    sort            integer NOT NULL DEFAULT 0,
    updated_at      timestamptz NOT NULL DEFAULT now()
);

INSERT INTO public.ai_models (code, title, kind, price_per_mtok, selectable, sort) VALUES
    ('gpt-5.4-nano',           'GPT',       'chat',      6100, true,  1),
    ('gemini-3.1-flash-lite',  'GEMINI',    'chat',      7600, true,  2),
    ('text-embedding-3-small', 'Эмбеддинги','embedding',  516, false, 3);

-- Личные настройки ИИ: выбранная модель, тумблеры функций и НЕОБЯЗАТЕЛЬНЫЙ свой
-- ключ (со своим адресом API) — на нём запросы идут мимо квоты токенов.
ALTER TABLE public.user_ai_settings
    ADD COLUMN IF NOT EXISTS api_base_url  text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS feat_assistant boolean NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS feat_notes     boolean NOT NULL DEFAULT true;

-- Прежний дефолт модели ассистента — пустая строка (брали модель компании).
UPDATE public.user_ai_settings SET model_chat = 'gpt-5.4-nano' WHERE model_chat = '';

-- Компания: ключа у неё больше нет, есть разрешение тратить токены создателя и
-- тумблеры её ИИ-функций.
ALTER TABLE public.companies
    ADD COLUMN IF NOT EXISTS ai_shared       boolean NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS ai_feat_search  boolean NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS ai_feat_tv_fact boolean NOT NULL DEFAULT true;

-- Прежние компанийные колонки ключа (ai_api_key_enc, ai_key_hint, ai_model_chat,
-- ai_model_embedding) НАМЕРЕННО остаются: на них ещё держится текущий aisvc.
-- Их удалит миграция, приезжающая вместе с переходом ИИ на платформенный ключ —
-- иначе схема опередила бы код и сервис перестал бы отвечать.

-- +goose Down
ALTER TABLE public.companies
    DROP COLUMN IF EXISTS ai_shared,
    DROP COLUMN IF EXISTS ai_feat_search,
    DROP COLUMN IF EXISTS ai_feat_tv_fact;

ALTER TABLE public.user_ai_settings
    DROP COLUMN IF EXISTS api_base_url,
    DROP COLUMN IF EXISTS feat_assistant,
    DROP COLUMN IF EXISTS feat_notes;

DROP TABLE IF EXISTS public.ai_models;
DROP TABLE IF EXISTS public.ai_platform_settings;
