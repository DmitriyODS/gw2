-- +goose Up
-- Код-приглашение компании для самостоятельного вступления по ссылке
-- (/join/<code>). Генерируют супер-админ или Руководитель компании; перегенерация
-- инвалидирует старую ссылку. NULL — приглашение не выдано.
ALTER TABLE public.companies ADD COLUMN invite_code character varying(32);
CREATE UNIQUE INDEX uq_companies_invite_code
    ON public.companies (invite_code) WHERE invite_code IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS public.uq_companies_invite_code;
ALTER TABLE public.companies DROP COLUMN IF EXISTS invite_code;
