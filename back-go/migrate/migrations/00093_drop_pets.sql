-- +goose Up
-- Питомцы-грувики и кудосы убраны из платформы целиком. Внешних ключей на эти
-- таблицы из остальной схемы нет, поэтому CASCADE задевает только их самих.
DROP TABLE IF EXISTS public.pet_bank_fund_donations,
                     public.pet_bank_funds,
                     public.pet_bank_goals,
                     public.pet_installments,
                     public.pet_kudos_ledger,
                     public.pet_kudos_seasonal,
                     public.pet_kudos_weekly,
                     public.pet_season_claims,
                     public.pet_shop_purchases,
                     public.pet_shop_items,
                     public.pet_strokes,
                     public.pet_activity_log,
                     public.pets CASCADE;

UPDATE public.companies SET settings = settings - 'uses_groove'
 WHERE settings ? 'uses_groove';

-- +goose Down
-- Необратимо: данные питомцев восстанавливаются только из резервной копии,
-- снятой до выката.
SELECT 1;
