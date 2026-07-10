-- +goose Up
-- Кудо-банк: выписка движений кудосов (леджер — приходы/расходы, переводы
-- между коллегами), сберегательный вклад под ежедневный процент и кредит
-- (тело + комиссия, один активный на питомца). Балансы вклада/долга — узкие
-- колонки pets, меняются только атомарными UPDATE (как престиж/домик).

ALTER TABLE public.pets ADD COLUMN bank_savings integer NOT NULL DEFAULT 0;
ALTER TABLE public.pets ADD COLUMN bank_savings_accrued_at timestamp with time zone;
ALTER TABLE public.pets ADD COLUMN bank_loan integer NOT NULL DEFAULT 0;

-- Леджер: каждая операция с кудосами (заработок, траты на питомца, покупки,
-- переводы, вклад/кредит). delta >0 — приход, <0 — расход; kind — источник;
-- counterparty — второй участник перевода.
CREATE TABLE public.pet_kudos_ledger (
    id bigserial PRIMARY KEY,
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    company_id integer NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    delta integer NOT NULL,
    kind text NOT NULL,
    counterparty_id integer REFERENCES public.users(id) ON DELETE SET NULL,
    comment text NOT NULL DEFAULT '',
    created_at timestamp with time zone NOT NULL DEFAULT now()
);

-- Выписка пользователя (keyset по id) и агрегаты компании (топ щедрости).
CREATE INDEX idx_pet_kudos_ledger_user ON public.pet_kudos_ledger (user_id, id DESC);
CREATE INDEX idx_pet_kudos_ledger_company_created
    ON public.pet_kudos_ledger (company_id, created_at DESC);

-- Ребаланс экономики: кудосы зарабатывались быстрее, чем тратились — витрина
-- магазина дорожает втрое (цены действий и домика — Go-константы petsvc,
-- подняты в том же релизе).
UPDATE public.pet_shop_items SET price_kudos = price_kudos * 3 WHERE price_kudos > 0;

-- +goose Down
UPDATE public.pet_shop_items SET price_kudos = price_kudos / 3 WHERE price_kudos > 0;
DROP TABLE IF EXISTS public.pet_kudos_ledger;
ALTER TABLE public.pets DROP COLUMN IF EXISTS bank_loan;
ALTER TABLE public.pets DROP COLUMN IF EXISTS bank_savings_accrued_at;
ALTER TABLE public.pets DROP COLUMN IF EXISTS bank_savings;
