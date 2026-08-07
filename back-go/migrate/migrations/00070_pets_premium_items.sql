-- +goose Up
-- Премиум-контент питомцев (скины, декор домика, товары) по тарифной линейке
-- доступен только на старшем тарифе. Признак — у самой позиции магазина:
-- ассортимент правится данными, без выката кода.
ALTER TABLE public.pet_shop_items
    ADD COLUMN IF NOT EXISTS premium boolean NOT NULL DEFAULT false;

-- Легендарные позиции — это и есть премиум-контент витрины.
UPDATE public.pet_shop_items SET premium = true WHERE rarity = 'legendary';

-- +goose Down
ALTER TABLE public.pet_shop_items DROP COLUMN IF EXISTS premium;
