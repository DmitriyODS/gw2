-- +goose Up
-- Отметка «портал просмотрен» (portalsvc): серверный счётчик непрочитанных
-- постов для бейджа в навигации, общий между устройствами пользователя.
-- Непрочитанные = посты компании с created_at позже seen_at (кроме своих);
-- заход на портал обновляет отметку.
CREATE TABLE public.portal_seen (
    user_id     bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    company_id  bigint NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    seen_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, company_id)
);

-- +goose Down
DROP TABLE public.portal_seen;
