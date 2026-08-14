-- +goose Up
-- Правовые документы: лицензионное соглашение, политика конфиденциальности и
-- согласие на обработку персональных данных (152-ФЗ).
--
-- В users лежит только ТЕКУЩЕЕ состояние (принятая редакция и когда) — по нему
-- гейт решает, пускать ли в приложение. Доказательство факта согласия обязан
-- хранить оператор (ч.1 ст.9 152-ФЗ), поэтому каждое принятие пишется ещё и
-- строкой журнала: редакция, перечень документов, время, адрес и клиент.
-- Журнал не переписывается — отзыв согласия ставит revoked_at, а новое
-- согласие добавляет строку: история должна быть восстановима целиком.
ALTER TABLE public.users
    ADD COLUMN legal_version text,
    ADD COLUMN legal_accepted_at timestamptz;

CREATE TABLE IF NOT EXISTS public.user_consents (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    -- Редакция документов, действовавшая в момент согласия (domain.LegalVersion).
    version text NOT NULL,
    -- Ключи принятых документов: ["license","privacy","consent"]. Список, а не
    -- набор колонок: состав документов со временем меняется, а старые строки
    -- обязаны сохранить ровно то, с чем человек соглашался.
    documents jsonb NOT NULL DEFAULT '[]'::jsonb,
    accepted_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    ip text,
    user_agent text
);

CREATE INDEX IF NOT EXISTS idx_user_consents_user ON public.user_consents (user_id, accepted_at DESC);

-- +goose Down
DROP TABLE IF EXISTS public.user_consents;

ALTER TABLE public.users
    DROP COLUMN IF EXISTS legal_version,
    DROP COLUMN IF EXISTS legal_accepted_at;
