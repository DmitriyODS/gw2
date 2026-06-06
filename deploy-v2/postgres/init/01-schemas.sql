-- Каждый сервис v4 владеет своей схемой в общем кластере. Сервис подключается
-- через DATABASE_URL с search_path=<своя_схема>,public и не имеет прав на
-- чужие таблицы — это эмулирует изоляцию микросервисов без отдельных кластеров.
--
-- Привилегии GRANT/REVOKE раздаются миграциями каждого сервиса (Этап 2+).
-- Здесь — только создание схем.

CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS users;
CREATE SCHEMA IF NOT EXISTS tasks;
CREATE SCHEMA IF NOT EXISTS messenger;
CREATE SCHEMA IF NOT EXISTS calls;
CREATE SCHEMA IF NOT EXISTS ai;
