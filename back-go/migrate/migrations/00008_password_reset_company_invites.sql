-- +goose Up
-- Восстановление пароля по email (одна активная заявка на пользователя) и
-- email-приглашения в компанию с ролью (одна активная заявка на пару
-- компания+email; перевыпуск перезаписывает токен). Чистятся при использовании.

CREATE TABLE public.password_resets (
    id           bigserial PRIMARY KEY,
    user_id      integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token        character varying(64) NOT NULL,
    expires_at   timestamp with time zone NOT NULL,
    last_sent_at timestamp with time zone NOT NULL DEFAULT now(),
    created_at   timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT password_resets_user_id_key UNIQUE (user_id),
    CONSTRAINT password_resets_token_key UNIQUE (token)
);

CREATE TABLE public.company_invites (
    id         bigserial PRIMARY KEY,
    company_id integer NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    email      character varying(255) NOT NULL,
    role_id    integer NOT NULL REFERENCES public.roles(id),
    token      character varying(64) NOT NULL,
    invited_by integer REFERENCES public.users(id) ON DELETE SET NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT company_invites_company_email_key UNIQUE (company_id, email),
    CONSTRAINT company_invites_token_key UNIQUE (token)
);

-- +goose Down
DROP TABLE public.company_invites;
DROP TABLE public.password_resets;
