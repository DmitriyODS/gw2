-- +goose Up
-- Развитие грувиков после максимальной формы: престиж-поколения (перерождение
-- «Легенды» с ростом счётчика поколений), сезонный трек наград (кудосы,
-- заработанные за календарный квартал, открывают пороги-награды) и домик
-- питомца (декор за кудосы — долгий сток валюты для прокачанных питомцев).

-- Поколение питомца: растёт при перерождении, никогда не сбрасывается.
ALTER TABLE public.pets ADD COLUMN generation integer NOT NULL DEFAULT 1;

-- Домик: купленный декор и расставленный (подмножество owned, лимит
-- проверяет сервис). Каталог декора — Go-константы petsvc (domain).
ALTER TABLE public.pets ADD COLUMN house_owned jsonb NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE public.pets ADD COLUMN house_placed jsonb NOT NULL DEFAULT '[]'::jsonb;

-- Кудосы, заработанные за сезон (календарный квартал МСК, ключ '2026-Q3') —
-- двигают сезонный трек; по образцу pet_kudos_weekly (трата не уменьшает).
CREATE TABLE public.pet_kudos_seasonal (
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    season text NOT NULL,
    amount integer NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, season)
);

-- Забранные награды трека: PK не даёт забрать один порог дважды.
CREATE TABLE public.pet_season_claims (
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    season text NOT NULL,
    threshold integer NOT NULL,
    claimed_at timestamp with time zone NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, season, threshold)
);

-- +goose Down
DROP TABLE IF EXISTS public.pet_season_claims;
DROP TABLE IF EXISTS public.pet_kudos_seasonal;
ALTER TABLE public.pets DROP COLUMN IF EXISTS house_placed;
ALTER TABLE public.pets DROP COLUMN IF EXISTS house_owned;
ALTER TABLE public.pets DROP COLUMN IF EXISTS generation;
