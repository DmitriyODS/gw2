-- +goose Up
-- Подтверждение email при самостоятельной регистрации. Новый аккаунт создаётся
-- с email_verified=false и не пускается в систему до подтверждения кодом/ссылкой.
-- Существующие аккаунты (и сотрудники, создаваемые администратором компании)
-- считаются подтверждёнными: бэкфилл в true, а пользователей-сотрудников вставляет
-- authsvc уже с email_verified=true (у них force-change пароля вместо верификации).

ALTER TABLE public.users ADD COLUMN email_verified boolean NOT NULL DEFAULT false;
UPDATE public.users SET email_verified = true;

-- Активные коды/ссылки подтверждения. Одна запись на пользователя (UNIQUE),
-- перевыпуск перезаписывает её. Чистится при успешном подтверждении.
CREATE TABLE public.email_verifications (
    id           bigserial PRIMARY KEY,
    user_id      integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    code         character varying(6) NOT NULL,
    token        character varying(64) NOT NULL,
    attempts     integer NOT NULL DEFAULT 0,
    expires_at   timestamp with time zone NOT NULL,
    last_sent_at timestamp with time zone NOT NULL DEFAULT now(),
    created_at   timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT email_verifications_user_id_key UNIQUE (user_id),
    CONSTRAINT email_verifications_token_key UNIQUE (token)
);

-- +goose Down
DROP TABLE public.email_verifications;
ALTER TABLE public.users DROP COLUMN email_verified;
