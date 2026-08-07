-- +goose Up
-- Реестр активных сессий пользователя («Авторизация и сессии» в профиле).
-- До этой таблицы refresh-токен был полностью stateless: отозвать конкретный
-- вход было нечем. Теперь refresh несёт id строки отсюда, и завершение сеанса —
-- это revoked_at: следующий refresh с таким токеном получает отказ, а access
-- доживает свои 15 минут.
--
-- last_seen_at двигает сам refresh (не чаще раза в 15 минут — по TTL access),
-- поэтому «последняя активность» точна с точностью до этого шага.
-- city заполняется асинхронно после входа (гео по IP, внешний сервис): вход не
-- ждёт сеть, а не определившийся город остаётся пустой строкой.
CREATE TABLE public.user_sessions (
    id           bigserial PRIMARY KEY,
    user_id      bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    revoked_at   timestamptz,

    -- Опознание устройства: platform — иконка карточки (mobile/desktop/web),
    -- client — обёртка это или браузер (app/web), device — модель или ОС.
    platform     varchar(16) NOT NULL DEFAULT 'web',
    client       varchar(16) NOT NULL DEFAULT 'web',
    device       varchar(120) NOT NULL DEFAULT '',
    user_agent   text NOT NULL DEFAULT '',
    ip           varchar(64) NOT NULL DEFAULT '',
    city         varchar(120) NOT NULL DEFAULT ''
);

-- Список профиля: живые сессии пользователя, свежие первыми.
CREATE INDEX user_sessions_active_idx ON public.user_sessions (user_id, last_seen_at DESC)
    WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS public.user_sessions;
