-- +goose Up
-- Чанковые загрузки — общий механизм ПЛАТФОРМЫ, а не одного раздела: файл
-- крупнее порога (pkg/chunkupload.Threshold) везде уезжает частями. Обрыв сети
-- на сотом мегабайте перестаёт означать «качай заново», а клиент показывает
-- честный прогресс.
--
-- Таблица одна на все сервисы (база у них общая): service отличает владельца
-- сессии, scope хранит его собственный контекст (реестр, папка, переписка).
-- Части лежат во временном префиксе хранилища и собираются в объект на finish.
CREATE TABLE public.upload_sessions (
    id         bigserial PRIMARY KEY,
    code       varchar(40) NOT NULL UNIQUE,
    service    varchar(32) NOT NULL,
    user_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    -- Чья квота тратится. Это НЕ обязательно загружающий: файл в чужом
    -- расшаренном реестре занимает место его владельца, а не гостя.
    -- Ровно одно из двух: компания-владелец либо пользователь (второе — 0).
    company_id    bigint NOT NULL DEFAULT 0,
    quota_user_id bigint NOT NULL DEFAULT 0,
    -- Контекст раздела: id реестра, папки диска, переписки. Смысл знает только
    -- сам сервис, поэтому строка.
    scope      varchar(64) NOT NULL DEFAULT '',
    file_name  varchar(255) NOT NULL,
    mime       varchar(120) NOT NULL DEFAULT '',
    total_size bigint NOT NULL,
    chunk_size integer NOT NULL,
    -- Сколько частей принято подряд: докачка продолжается со следующей.
    received   integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
-- Уборка брошенных сессий (человек закрыл вкладку на середине).
CREATE INDEX upload_sessions_stale_idx ON public.upload_sessions (updated_at);
CREATE INDEX upload_sessions_user_idx ON public.upload_sessions (user_id, service);

-- +goose Down
DROP TABLE IF EXISTS public.upload_sessions;
