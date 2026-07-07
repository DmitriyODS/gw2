-- +goose Up
-- «Приключение питомца» (petsvc): appointment-механика — владелец отправляет
-- питомца в приключение на 2–4 часа, возврат фиксируется лениво на GET
-- владельца и начисляет вариативную награду. Пока adventure_until в будущем —
-- платные действия недоступны.
ALTER TABLE pets ADD COLUMN adventure_until TIMESTAMPTZ;
ALTER TABLE pets ADD COLUMN adventure_place TEXT;

-- +goose Down
ALTER TABLE pets DROP COLUMN adventure_place;
ALTER TABLE pets DROP COLUMN adventure_until;
