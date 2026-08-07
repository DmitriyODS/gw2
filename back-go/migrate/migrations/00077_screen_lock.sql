-- +goose Up
-- Экран блокировки: пока человека нет за устройством, приложение закрывается
-- пин-кодом, а сессия при этом остаётся живой (выход из аккаунта потерял бы
-- открытые окна и черновики).
--
-- Пин хранится ХЕШЕМ (pgcrypto, как пароли): он короткий, и утечка таблицы не
-- должна давать доступ к аккаунтам. Проверка идёт на сервере, а не в браузере —
-- иначе снять блокировку можно было бы правкой localStorage.
ALTER TABLE public.users
    ADD COLUMN lock_pin_hash text,
    -- Через сколько минут бездействия запирать; NULL — только вручную.
    ADD COLUMN lock_after_min integer;

-- +goose Down
ALTER TABLE public.users
    DROP COLUMN IF EXISTS lock_pin_hash,
    DROP COLUMN IF EXISTS lock_after_min;
