-- +goose Up
-- Возврат облика «Пчёлка/Шмель» (bee) в магазин: строка из baseline-сида
-- (00018) пропала в проде. Идемпотентно — если вид на месте, ничего не меняем.
INSERT INTO public.pet_shop_items (key, kind, rarity, price_kudos, unlock_kind)
VALUES ('bee', 'species', 'epic', 160, 'shop')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
-- Down намеренно пуст: bee — часть исходного ассортимента (00018), удаление
-- его этой миграцией было бы регрессией.
SELECT 1;
