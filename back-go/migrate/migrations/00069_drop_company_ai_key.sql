-- +goose Up
-- Ключи компаний больше не используются: ИИ работает на ПЛАТФОРМЕННОМ ключе
-- (ai_platform_settings), а компания лишь включает возможности и разрешает
-- тратить токены своего создателя. Колонки удаляем отдельной миграцией — после
-- того, как код перестал их читать (миграция 00068 их намеренно оставила).
ALTER TABLE public.companies
    DROP COLUMN IF EXISTS ai_api_key_enc,
    DROP COLUMN IF EXISTS ai_key_hint,
    DROP COLUMN IF EXISTS ai_model_chat,
    DROP COLUMN IF EXISTS ai_model_embedding;

-- +goose Down
ALTER TABLE public.companies
    ADD COLUMN IF NOT EXISTS ai_api_key_enc     bytea,
    ADD COLUMN IF NOT EXISTS ai_key_hint        varchar(16),
    ADD COLUMN IF NOT EXISTS ai_model_chat      varchar(64) NOT NULL DEFAULT 'gpt-4o-mini',
    ADD COLUMN IF NOT EXISTS ai_model_embedding varchar(64) NOT NULL DEFAULT 'text-embedding-3-small';
