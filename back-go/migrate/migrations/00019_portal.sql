-- +goose Up
-- Раздел «Корпоративный портал» (portalsvc): посты компании с комментариями,
-- реакциями, закреплением и тематическими разделами. Полностью независим от
-- питомцев-грувиков (petsvc) — только контент/коллаборация. Топики ведёт
-- администратор компании, посты/комментарии/реакции — любой участник.
-- Закрепление — автор поста или администратор, лимит 10 закреплённых на
-- компанию проверяется в сервисе (аналог SharePoint boost-лимита).

CREATE TABLE public.portal_topics (
    id          bigserial PRIMARY KEY,
    company_id  bigint NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    name        varchar(120) NOT NULL,
    color       text,
    icon        text,
    created_by  bigint NOT NULL REFERENCES public.users(id),
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX portal_topics_company_idx ON public.portal_topics (company_id);

CREATE TABLE public.portal_posts (
    id          bigserial PRIMARY KEY,
    company_id  bigint NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    topic_id    bigint REFERENCES public.portal_topics(id) ON DELETE SET NULL,
    author_id   bigint NOT NULL REFERENCES public.users(id),
    title       text,
    body        text NOT NULL,
    pinned_at   timestamptz,
    pinned_by   bigint REFERENCES public.users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
-- Список постов: закреплённые сверху (pinned_at DESC NULLS LAST), затем
-- хронология — индекс покрывает основную сортировку выборки компании.
CREATE INDEX portal_posts_company_idx ON public.portal_posts (company_id, pinned_at DESC NULLS LAST, created_at DESC);
CREATE INDEX portal_posts_topic_idx ON public.portal_posts (topic_id);

-- Сквозной ILIKE-поиск по названию+телу поста — триграммный GIN-индекс, как у
-- registry/calendar (pg_trgm включён в baseline).
CREATE INDEX portal_posts_search_idx ON public.portal_posts
    USING gin ((coalesce(title, '') || ' ' || body) public.gin_trgm_ops);

CREATE TABLE public.portal_attachments (
    id          bigserial PRIMARY KEY,
    post_id     bigint NOT NULL REFERENCES public.portal_posts(id) ON DELETE CASCADE,
    file_path   text NOT NULL,
    name        text NOT NULL,
    size        bigint NOT NULL,
    mime        text,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX portal_attachments_post_idx ON public.portal_attachments (post_id);

CREATE TABLE public.portal_comments (
    id          bigserial PRIMARY KEY,
    post_id     bigint NOT NULL REFERENCES public.portal_posts(id) ON DELETE CASCADE,
    author_id   bigint NOT NULL REFERENCES public.users(id),
    text        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX portal_comments_post_idx ON public.portal_comments (post_id, created_at);

CREATE TABLE public.portal_reactions (
    post_id     bigint NOT NULL REFERENCES public.portal_posts(id) ON DELETE CASCADE,
    user_id     bigint NOT NULL REFERENCES public.users(id),
    emoji       text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, user_id, emoji)
);

-- +goose Down
DROP TABLE public.portal_reactions;
DROP TABLE public.portal_comments;
DROP TABLE public.portal_attachments;
DROP TABLE public.portal_posts;
DROP TABLE public.portal_topics;
