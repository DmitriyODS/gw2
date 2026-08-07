-- +goose Up
-- Несколько аккаунтов YouGile у одного пользователя с переключением между
-- ними: люди работают в разных пространствах YouGile, и раньше подключение
-- второго вытесняло первое (UNIQUE по user_id).
--
-- Активен ровно один аккаунт: именно его ключ идёт в импорт/экспорт карточек.
-- «Ровно один» держит частичный уникальный индекс, а не флаг в приложении.
ALTER TABLE public.user_yougile_accounts DROP CONSTRAINT IF EXISTS uq_user_yg_account;

-- Один аккаунт на пространство YouGile: повторное подключение того же
-- пространства обновляет ключ, а не плодит копии.
ALTER TABLE public.user_yougile_accounts
    ADD CONSTRAINT uq_user_yg_company UNIQUE (user_id, yg_company_id);

ALTER TABLE public.user_yougile_accounts
    ADD COLUMN is_active boolean NOT NULL DEFAULT TRUE;

CREATE UNIQUE INDEX user_yougile_accounts_active_idx
    ON public.user_yougile_accounts (user_id) WHERE is_active;

-- Существующие подключения остаются активными (их по одному на человека).

-- +goose Down
DROP INDEX IF EXISTS public.user_yougile_accounts_active_idx;
ALTER TABLE public.user_yougile_accounts DROP COLUMN IF EXISTS is_active;
ALTER TABLE public.user_yougile_accounts DROP CONSTRAINT IF EXISTS uq_user_yg_company;
ALTER TABLE public.user_yougile_accounts
    ADD CONSTRAINT uq_user_yg_account UNIQUE (user_id);
