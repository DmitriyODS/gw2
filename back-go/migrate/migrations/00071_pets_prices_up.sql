-- +goose Up
-- Ребаланс экономики грувиков: кудосы копились быстрее, чем тратились, и
-- витрина переставала быть целью. Поднимаем цены магазина в полтора раза —
-- ровно на столько же подорожали уход и декор домика (domain/consts.go).
-- Округляем до десятков, чтобы ценники оставались читаемыми.
UPDATE public.pet_shop_items
   SET price_kudos = GREATEST(10, round(price_kudos * 1.5 / 10) * 10)
 WHERE price_kudos > 0;

-- +goose Down
UPDATE public.pet_shop_items
   SET price_kudos = GREATEST(10, round(price_kudos / 1.5 / 10) * 10)
 WHERE price_kudos > 0;
