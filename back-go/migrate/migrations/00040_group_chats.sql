-- +goose Up
-- Групповые чаты. Диалог-пара (user_a<user_b) не масштабируется на N участников,
-- поэтому для групп заводится таблица участников conversation_members с
-- watermark-прочтением (last_read_message_id) и личными состояниями. У группы
-- user_a_id/user_b_id = NULL, company_id = NULL (участники — любые пользователи
-- платформы, cross-company).
ALTER TABLE public.conversations ADD COLUMN is_group boolean DEFAULT false NOT NULL;
ALTER TABLE public.conversations ADD COLUMN title varchar(120);
ALTER TABLE public.conversations ADD COLUMN avatar_path text;
ALTER TABLE public.conversations ADD COLUMN created_by integer REFERENCES public.users(id) ON DELETE SET NULL;
ALTER TABLE public.conversations ADD COLUMN invite_code varchar(64);
CREATE UNIQUE INDEX uq_conversation_invite_code ON public.conversations (invite_code) WHERE invite_code IS NOT NULL;

-- Обновляем инвариант структуры диалога: dev-чат, группа или пара.
ALTER TABLE public.conversations DROP CONSTRAINT IF EXISTS ck_conversation_pair_order;
ALTER TABLE public.conversations ADD CONSTRAINT ck_conversation_pair_order CHECK (
    (is_dev_chat AND NOT is_group AND user_a_id IS NOT NULL AND user_b_id IS NULL)
    OR (is_group AND NOT is_dev_chat AND user_a_id IS NULL AND user_b_id IS NULL)
    OR (NOT is_dev_chat AND NOT is_group AND user_a_id IS NOT NULL AND user_b_id IS NOT NULL AND user_a_id < user_b_id)
);

CREATE TABLE public.conversation_members (
    conversation_id integer NOT NULL REFERENCES public.conversations(id) ON DELETE CASCADE,
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    role varchar(16) DEFAULT 'member' NOT NULL,          -- owner | admin | member
    joined_at timestamp with time zone DEFAULT now() NOT NULL,
    last_read_message_id integer,                        -- watermark непрочитанных + база «кто прочитал»
    last_read_at timestamp with time zone,
    pinned_at timestamp with time zone,                  -- личное закрепление группы
    hidden_at timestamp with time zone,                  -- скрыл группу у себя (не выходя)
    muted boolean DEFAULT false NOT NULL,
    -- гранулярные права для role='admin' (у owner всё, у member ничего):
    can_manage_members boolean DEFAULT true NOT NULL,
    can_edit_info boolean DEFAULT true NOT NULL,
    can_pin_messages boolean DEFAULT true NOT NULL,
    PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX idx_conv_members_user ON public.conversation_members (user_id);

-- +goose Down
DROP TABLE public.conversation_members;
ALTER TABLE public.conversations DROP CONSTRAINT IF EXISTS ck_conversation_pair_order;
ALTER TABLE public.conversations ADD CONSTRAINT ck_conversation_pair_order CHECK (
    (is_dev_chat AND user_a_id IS NOT NULL AND user_b_id IS NULL)
    OR (NOT is_dev_chat AND user_a_id IS NOT NULL AND user_b_id IS NOT NULL AND user_a_id < user_b_id)
);
DROP INDEX IF EXISTS uq_conversation_invite_code;
ALTER TABLE public.conversations DROP COLUMN invite_code;
ALTER TABLE public.conversations DROP COLUMN created_by;
ALTER TABLE public.conversations DROP COLUMN avatar_path;
ALTER TABLE public.conversations DROP COLUMN title;
ALTER TABLE public.conversations DROP COLUMN is_group;
