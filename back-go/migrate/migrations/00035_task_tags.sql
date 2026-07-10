-- +goose Up
-- Теги задач: справочник компании (имя + цвет из палитры --tag-*) и связка
-- many-to-many с задачами. Теги ОБЩИЕ для компании (в отличие от личного
-- цвета карточки user_task_colors) — транслируются в сокет-события.

CREATE TABLE public.tags (
    id bigserial PRIMARY KEY,
    company_id integer NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    name character varying(64) NOT NULL,
    color character varying(16) NOT NULL DEFAULT 'blue'
);
CREATE UNIQUE INDEX idx_tags_company_name ON public.tags (company_id, lower(name));

CREATE TABLE public.task_tags (
    task_id integer NOT NULL REFERENCES public.tasks(id) ON DELETE CASCADE,
    tag_id bigint NOT NULL REFERENCES public.tags(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, tag_id)
);
-- Фильтр списка задач по тегу.
CREATE INDEX idx_task_tags_tag ON public.task_tags (tag_id);

-- +goose Down
DROP TABLE IF EXISTS public.task_tags;
DROP TABLE IF EXISTS public.tags;
