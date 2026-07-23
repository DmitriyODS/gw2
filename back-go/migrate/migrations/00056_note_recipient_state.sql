-- +goose Up
-- Личный оверлей адресата шаринга заметок/папок: позволяет тому, с кем
-- поделились, разложить чужие заметки/папки по СВОИМ папкам и отправить чужую
-- заметку в личный архив — не затрагивая владельца (у него свои folder_id/
-- archived). Наличие строки = адресат «взял размещение под контроль»: без строки
-- элемент остаётся в разделе «Поделились со мной».

-- Размещение расшаренной мне ЗАМЕТКИ: folder_id — моя папка (NULL — мой корень),
-- archived — личный архив.
CREATE TABLE note_recipient_state (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    note_id    BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    folder_id  BIGINT REFERENCES note_folders(id) ON DELETE SET NULL,
    archived   BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, note_id)
);
CREATE INDEX idx_note_recipient_state_folder ON note_recipient_state(user_id, folder_id) WHERE NOT archived;

-- Размещение расшаренной мне ПАПКИ-корня под моей папкой: parent_id — моя папка
-- (NULL — мой корень), archived — личный архив (на будущее, симметрия с заметками).
CREATE TABLE folder_recipient_state (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    folder_id  BIGINT NOT NULL REFERENCES note_folders(id) ON DELETE CASCADE,
    parent_id  BIGINT REFERENCES note_folders(id) ON DELETE SET NULL,
    archived   BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, folder_id)
);
CREATE INDEX idx_folder_recipient_state_parent ON folder_recipient_state(user_id, parent_id) WHERE NOT archived;

-- +goose Down
DROP TABLE IF EXISTS folder_recipient_state;
DROP TABLE IF EXISTS note_recipient_state;
