-- +goose Up
-- Хештеги постов портала (как в соцсетях): парсятся сервером из тела поста
-- (#релиз, #hr), кликабельны и фильтруют ленту. Тег хранится нормализованным
-- (lower). company_id денормализован — топ популярных тегов компании и фильтр
-- ленты по тегу считаются без join к portal_posts.
CREATE TABLE public.portal_post_tags (
    post_id    bigint NOT NULL REFERENCES public.portal_posts(id) ON DELETE CASCADE,
    company_id bigint NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    tag        text   NOT NULL,
    PRIMARY KEY (post_id, tag)
);
-- Фильтр ленты по тегу и топ популярных тегов компании.
CREATE INDEX portal_post_tags_company_tag_idx ON public.portal_post_tags (company_id, tag);

-- +goose Down
DROP TABLE IF EXISTS public.portal_post_tags;
