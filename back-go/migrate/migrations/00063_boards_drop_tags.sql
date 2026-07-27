-- +goose Up
-- У досок теги-метки не прижились: организация раздела — иерархические папки,
-- второй параллельный способ раскладки только путал. Заметок это не касается,
-- там теги остаются (note_tags).
DROP TABLE IF EXISTS public.board_tag_items;
DROP TABLE IF EXISTS public.board_tags;

-- +goose Down
CREATE TABLE public.board_tags (
    id         bigserial PRIMARY KEY,
    owner_id   bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    name       varchar(100) NOT NULL,
    color      varchar(16) NOT NULL DEFAULT '',
    position   integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX board_tags_owner_idx ON public.board_tags (owner_id, position);

CREATE TABLE public.board_tag_items (
    board_id bigint NOT NULL REFERENCES public.boards(id) ON DELETE CASCADE,
    tag_id   bigint NOT NULL REFERENCES public.board_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (board_id, tag_id)
);
CREATE INDEX board_tag_items_tag_idx ON public.board_tag_items (tag_id);
