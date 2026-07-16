-- +goose Up
-- Потребности грувика и виды болезней.
--
-- Потребности — четыре шкалы 0..100, тающие со временем (пересчёт ленивый, по
-- needs_at, без фонового цикла — как возврат из приключения). Пустая шкала
-- вводит питомца в СВОЮ болезнь, поэтому вид болезни теперь хранится явно:
-- ailment (NULL — здоров) рядом с прежними sick_since/recovery. Инвариант:
-- ailment IS NOT NULL ⟺ sick_since IS NOT NULL.
ALTER TABLE public.pets
    ADD COLUMN need_satiety smallint NOT NULL DEFAULT 100,
    ADD COLUMN need_energy  smallint NOT NULL DEFAULT 100,
    ADD COLUMN need_hygiene smallint NOT NULL DEFAULT 100,
    ADD COLUMN need_social  smallint NOT NULL DEFAULT 100,
    ADD COLUMN needs_at     timestamp with time zone DEFAULT now() NOT NULL,
    ADD COLUMN ailment      text;

-- Уже болеющие питомцы болели единственной прежней болезнью — простоем в
-- работе («хандра»).
UPDATE public.pets SET ailment = 'blues' WHERE sick_since IS NOT NULL;

-- +goose Down
ALTER TABLE public.pets
    DROP COLUMN need_satiety,
    DROP COLUMN need_energy,
    DROP COLUMN need_hygiene,
    DROP COLUMN need_social,
    DROP COLUMN needs_at,
    DROP COLUMN ailment;
