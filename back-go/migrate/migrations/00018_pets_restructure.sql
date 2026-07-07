-- +goose Up
-- Разделение «Моего Groove» на независимые сущности (см. план
-- «Переосмысление Мой Groove»): питомцы-грувики становятся чистой
-- гейм-механикой без публичной ленты/рейда/wrapped/AI-чата/погоды.

-- Публичная доска признания, недельные рейды, локации для погодных
-- реплик — убираются целиком вместе с историческими данными.
DROP TABLE IF EXISTS public.feed_comments, public.feed_reactions,
    public.feed_events, public.groove_raids, public.user_locations CASCADE;

-- «Грувы» → «кудосы»: валюта питомца остаётся, меняется только имя.
ALTER TABLE public.pets RENAME COLUMN beans TO kudos;

-- Магазин переезжает из Go-констант в БД, чтобы ассортимент можно было
-- ротировать без деплоя (постоянные/сезонные/лимитированные/достижимые
-- товары одновременно).
CREATE TABLE public.pet_shop_items (
    id bigserial PRIMARY KEY,
    key text NOT NULL UNIQUE,
    kind text NOT NULL CHECK (kind IN ('skin', 'accessory', 'species')),
    rarity text NOT NULL CHECK (rarity IN ('common', 'rare', 'epic', 'legendary')),
    price_kudos integer NOT NULL DEFAULT 0,
    unlock_kind text NOT NULL DEFAULT 'shop' CHECK (unlock_kind IN ('shop', 'achievement')),
    achievement_key text,
    limited_quota integer,
    active_from timestamp with time zone,
    active_to timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE TABLE public.pet_shop_purchases (
    id bigserial PRIMARY KEY,
    item_id bigint NOT NULL REFERENCES public.pet_shop_items(id) ON DELETE CASCADE,
    company_id integer NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    purchased_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE INDEX idx_pet_shop_purchases_item_company
    ON public.pet_shop_purchases (item_id, company_id);

-- Приватная история активности питомца — узкая замена публичной ленты,
-- видна только владельцу (вкладка «История» в модалке питомца).
CREATE TABLE public.pet_activity_log (
    id bigserial PRIMARY KEY,
    pet_user_id integer NOT NULL REFERENCES public.pets(user_id) ON DELETE CASCADE,
    kind text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE INDEX idx_pet_activity_log_pet_created
    ON public.pet_activity_log (pet_user_id, created_at DESC);

-- Реанимация pet_strokes (была мёртвой таблицей): дневной лимит
-- StrokeDailyMaxPerPet считается по количеству строк за день на пару
-- «гладящий → питомец» — снимаем старое UNIQUE(pet_user_id, user_id, day)
-- (оно держало жёсткий лимит «раз в день суммарно») в пользу обычного
-- индекса для быстрого подсчёта.
ALTER TABLE public.pet_strokes DROP CONSTRAINT IF EXISTS uq_pet_stroke_day;
CREATE INDEX IF NOT EXISTS idx_pet_strokes_pet_stroker_day
    ON public.pet_strokes (pet_user_id, user_id, day);

-- Счётчик признания рейтинга: кудосы, начисленные с начала текущей ISO-
-- недели (раньше это была публичная кудос-реакция ленты — теперь просто
-- сумма экономических начислений; отдельный маленький счётчик проще, чем
-- пересчитывать его каждый раз из общей истории начислений).
CREATE TABLE public.pet_kudos_weekly (
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    iso_year integer NOT NULL,
    iso_week integer NOT NULL,
    amount integer NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, iso_year, iso_week)
);

-- Сид ассортимента: перенос прежних Go-констант ShopPrices/SeasonalItems/
-- RareItems/SpeciesShop как строк (структура важнее полноты — фактический
-- ассортимент дальше правится через данные, не деплой).
INSERT INTO public.pet_shop_items (key, kind, rarity, price_kudos, unlock_kind) VALUES
    ('party', 'accessory', 'common', 30, 'shop'),
    ('cap', 'accessory', 'common', 40, 'shop'),
    ('bow', 'accessory', 'common', 40, 'shop'),
    ('scarf', 'accessory', 'common', 50, 'shop'),
    ('tie', 'accessory', 'common', 50, 'shop'),
    ('glasses', 'accessory', 'rare', 60, 'shop'),
    ('headphones', 'accessory', 'rare', 60, 'shop'),
    ('mask', 'accessory', 'rare', 70, 'shop'),
    ('tophat', 'accessory', 'rare', 80, 'shop'),
    ('medal', 'accessory', 'epic', 90, 'shop'),
    ('crown', 'accessory', 'legendary', 200, 'shop'),
    ('cat', 'species', 'common', 80, 'shop'),
    ('dog', 'species', 'common', 80, 'shop'),
    ('rabbit', 'species', 'common', 80, 'shop'),
    ('frog', 'species', 'common', 80, 'shop'),
    ('hamster', 'species', 'common', 90, 'shop'),
    ('chick', 'species', 'common', 100, 'shop'),
    ('monkey', 'species', 'rare', 100, 'shop'),
    ('hedgehog', 'species', 'rare', 110, 'shop'),
    ('panda', 'species', 'rare', 120, 'shop'),
    ('koala', 'species', 'rare', 130, 'shop'),
    ('tiger', 'species', 'epic', 140, 'shop'),
    ('bear', 'species', 'epic', 140, 'shop'),
    ('penguin', 'species', 'epic', 140, 'shop'),
    ('deer', 'species', 'epic', 150, 'shop'),
    ('bee', 'species', 'epic', 160, 'shop'),
    ('octopus', 'species', 'epic', 170, 'shop'),
    ('wolf', 'species', 'legendary', 180, 'shop'),
    ('lion', 'species', 'legendary', 200, 'shop'),
    ('dolphin', 'species', 'legendary', 200, 'shop'),
    ('whale', 'species', 'legendary', 230, 'shop'),
    ('unicorn', 'species', 'legendary', 250, 'shop'),
    ('dragon', 'species', 'legendary', 250, 'shop');

-- Редкие праздничные товары: окно вокруг ближайшей даты события (key
-- уникален — следующий год заводится новой строкой вручную/сидом при
-- ротации витрины, не задачей этой миграции).
INSERT INTO public.pet_shop_items (key, kind, rarity, price_kudos, unlock_kind, active_from, active_to) VALUES
    ('fireworks', 'accessory', 'epic', 80, 'shop', '2026-12-25', '2027-01-08'),
    ('love', 'accessory', 'epic', 60, 'shop', '2027-02-10', '2027-02-16'),
    ('shamrock', 'accessory', 'epic', 60, 'shop', '2027-03-14', '2027-03-20'),
    ('rocket', 'accessory', 'epic', 70, 'shop', '2027-04-08', '2027-04-16'),
    ('graduation', 'accessory', 'epic', 75, 'shop', '2027-06-20', '2027-07-10');

-- Мессенджер: pet-чат убран вместе с ИИ-персонажем питомца — существующие
-- диалоги-чаты с питомцем удаляются, флаг больше не нужен.
DELETE FROM public.conversations WHERE is_pet_chat = TRUE;
ALTER TABLE public.conversations DROP CONSTRAINT IF EXISTS ck_conversation_pair_order;
ALTER TABLE public.conversations DROP COLUMN is_pet_chat;
ALTER TABLE public.conversations ADD CONSTRAINT ck_conversation_pair_order CHECK (
    (is_dev_chat AND user_a_id IS NOT NULL AND user_b_id IS NULL)
    OR (NOT is_dev_chat AND user_a_id IS NOT NULL AND user_b_id IS NOT NULL AND user_a_id < user_b_id)
);

-- +goose Down
ALTER TABLE public.conversations DROP CONSTRAINT IF EXISTS ck_conversation_pair_order;
ALTER TABLE public.conversations ADD COLUMN is_pet_chat boolean DEFAULT false NOT NULL;
ALTER TABLE public.conversations ADD CONSTRAINT ck_conversation_pair_order CHECK (
    (((is_dev_chat OR is_pet_chat) AND (NOT (is_dev_chat AND is_pet_chat))
        AND (user_a_id IS NOT NULL) AND (user_b_id IS NULL))
    OR ((NOT is_dev_chat) AND (NOT is_pet_chat) AND (user_a_id IS NOT NULL)
        AND (user_b_id IS NOT NULL) AND (user_a_id < user_b_id)))
);

DROP TABLE IF EXISTS public.pet_kudos_weekly;
DROP INDEX IF EXISTS public.idx_pet_strokes_pet_stroker_day;
ALTER TABLE public.pet_strokes ADD CONSTRAINT uq_pet_stroke_day
    UNIQUE (pet_user_id, user_id, day);
DROP TABLE IF EXISTS public.pet_activity_log;
DROP TABLE IF EXISTS public.pet_shop_purchases;
DROP TABLE IF EXISTS public.pet_shop_items;
ALTER TABLE public.pets RENAME COLUMN kudos TO beans;
