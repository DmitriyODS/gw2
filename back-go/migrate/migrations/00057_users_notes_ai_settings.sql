-- +goose Up
-- Персональные настройки ИИ в заметках (переносятся между устройствами): по
-- кнопке корректировать орфографию/пунктуацию и дописывать текст. Обе выключены
-- по умолчанию.
ALTER TABLE users
    ADD COLUMN notes_ai_proofread    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN notes_ai_autocomplete BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE users
    DROP COLUMN IF EXISTS notes_ai_proofread,
    DROP COLUMN IF EXISTS notes_ai_autocomplete;
