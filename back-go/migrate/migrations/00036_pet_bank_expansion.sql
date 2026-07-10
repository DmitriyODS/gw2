-- +goose Up
-- Кудо-банк 2.0: копилки-цели (личные суб-счета «коплю на мечту», без
-- процента — процент только у вклада) и благотворительные сборы компании
-- (общая цель, куда скидываются коллеги; собранное считается потраченным —
-- возвратов нет). Кошелёк по-прежнему pets.kudos, движения — атомарные
-- UPDATE с guard'ами + записи леджера в той же транзакции.

-- Тема комнаты грувика: ключ градиентного пресета (визуал — на фронте),
-- видна коллегам в домике и мини-игре поглаживания.
ALTER TABLE public.pets ADD COLUMN house_theme text NOT NULL DEFAULT 'cozy';

-- Позиция самого грувика в сцене комнаты (проценты; NULL — место по умолчанию).
ALTER TABLE public.pets ADD COLUMN house_pet_x real;
ALTER TABLE public.pets ADD COLUMN house_pet_y real;

CREATE TABLE public.pet_bank_goals (
    id bigserial PRIMARY KEY,
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    company_id integer NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    title text NOT NULL,
    emoji text NOT NULL DEFAULT '🎯',
    target integer NOT NULL,
    saved integer NOT NULL DEFAULT 0,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    achieved_at timestamp with time zone
);

CREATE INDEX idx_pet_bank_goals_user ON public.pet_bank_goals (user_id, id);

CREATE TABLE public.pet_bank_funds (
    id bigserial PRIMARY KEY,
    company_id integer NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    created_by integer REFERENCES public.users(id) ON DELETE SET NULL,
    title text NOT NULL,
    description text NOT NULL DEFAULT '',
    emoji text NOT NULL DEFAULT '💝',
    target integer NOT NULL,
    collected integer NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'active', -- active | done | closed
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    finished_at timestamp with time zone
);

CREATE INDEX idx_pet_bank_funds_company ON public.pet_bank_funds (company_id, status, id DESC);

CREATE TABLE public.pet_bank_fund_donations (
    id bigserial PRIMARY KEY,
    fund_id bigint NOT NULL REFERENCES public.pet_bank_funds(id) ON DELETE CASCADE,
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    amount integer NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE INDEX idx_pet_bank_fund_donations_fund ON public.pet_bank_fund_donations (fund_id, user_id);

-- +goose Down
DROP TABLE IF EXISTS public.pet_bank_fund_donations;
DROP TABLE IF EXISTS public.pet_bank_funds;
DROP TABLE IF EXISTS public.pet_bank_goals;
ALTER TABLE public.pets DROP COLUMN IF EXISTS house_pet_y;
ALTER TABLE public.pets DROP COLUMN IF EXISTS house_pet_x;
ALTER TABLE public.pets DROP COLUMN IF EXISTS house_theme;
