-- +goose Up
-- Магазин обликов: покупной вид «Лисёнок» (природный fox — «Универсал» —
-- остаётся отдельным бесплатным видом эволюции).
INSERT INTO public.pet_shop_items (key, kind, rarity, price_kudos, unlock_kind)
VALUES ('foxy', 'species', 'rare', 110, 'shop')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DELETE FROM public.pet_shop_items WHERE key = 'foxy';
