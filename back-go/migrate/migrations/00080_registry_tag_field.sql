-- +goose Up
-- Теги реестра: администратор компании выбирает ОДНО списковое поле, значения
-- которого становятся чипами-фильтрами над таблицей записей. Удалили поле —
-- теги отключаются сами (SET NULL), а не оставляют ссылку в никуда.
ALTER TABLE registries
    ADD COLUMN tag_field_id bigint REFERENCES registry_fields (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE registries DROP COLUMN tag_field_id;
