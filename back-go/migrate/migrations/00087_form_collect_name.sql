-- +goose Up
-- Настройка «спрашивать имя» у формы: гость по внешней ссылке представляется
-- сам, и не каждой анкете это нужно (у анонимного опроса имя только мешает).
-- Умолчание TRUE — прежнее поведение сохраняется.
ALTER TABLE public.forms
    ADD COLUMN collect_name boolean NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE public.forms DROP COLUMN IF EXISTS collect_name;
