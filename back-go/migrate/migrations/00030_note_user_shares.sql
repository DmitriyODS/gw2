-- +goose Up
-- Адресный шаринг заметок: заметка открыта конкретному пользователю платформы
-- (вкладка «Поделились»). can_edit — адресату разрешено править title/doc
-- (иначе только чтение). Пара (note_id, user_id) уникальна — один доступ на
-- адресата, повторная выдача идемпотентно меняет право.
CREATE TABLE public.note_user_shares (
    note_id    bigint NOT NULL REFERENCES public.notes(id) ON DELETE CASCADE,
    user_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    can_edit   boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (note_id, user_id)
);
-- Выборка «поделились со мной» — по адресату.
CREATE INDEX note_user_shares_user_idx ON public.note_user_shares (user_id);

-- +goose Down
DROP TABLE IF EXISTS public.note_user_shares;
