-- +goose Up
-- Публичные ссылки на реестр (read-only доступ без авторизации). Код в URL —
-- capability: кто знает код, видит таблицу реестра (просмотр/карточки/экспорт),
-- но не может редактировать. Отзыв ссылки = удаление строки.
CREATE TABLE public.registry_shares (
    id          bigserial PRIMARY KEY,
    registry_id bigint NOT NULL REFERENCES public.registries(id) ON DELETE CASCADE,
    code        varchar(40) NOT NULL UNIQUE,
    created_by  bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX registry_shares_registry_idx ON public.registry_shares (registry_id);

-- +goose Down
DROP TABLE IF EXISTS public.registry_shares;
