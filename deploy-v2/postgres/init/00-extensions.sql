-- Расширения PostgreSQL, нужные всем сервисам v4.
-- Выполняется один раз при первом старте кластера (docker-entrypoint-initdb.d).
--
-- ВАЖНО: pgcrypto и citext подтянутся из postgres:18-alpine; pgvector нужен
-- образ с предустановленным расширением — мы используем pgvector/pgvector:pg18.

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS vector;
