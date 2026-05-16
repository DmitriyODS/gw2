-- Расширение для хеширования паролей
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Роль с полным доступом (BIGINT MAX = все биты = 1)
INSERT INTO roles (name, access)
VALUES ('Администратор', 9223372036854775807);

-- Первый пользователь: логин admin / пароль admin
-- is_default_pass = TRUE → принудительная смена при первом входе
INSERT INTO users (fio, login, hash_password, role_id, is_default_pass)
VALUES (
    'Администратор',
    'admin',
    crypt('admin', gen_salt('bf')),
    1,
    TRUE
);
