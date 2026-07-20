-- +goose Up
-- Режим «в отпуске»: пользователь сам включает в профиле. Пока включён —
-- нельзя создавать/редактировать задачи и запускать юниты (гарды в tasksvc),
-- а грувик тоже уходит в отпуск (petsvc замораживает потребности и болезни).
ALTER TABLE users ADD COLUMN on_vacation BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS on_vacation;
