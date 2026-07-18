-- +goose Up
-- Семантический (ИИ) поиск по заметкам — по образцу task_embeddings задач.
-- Эмбеддинг заметки (pgvector, модель text-embedding-3-small, 1536). Скоуп —
-- по владельцу (owner_id), как и сами заметки. Векторизация берётся из общего
-- aisvc.Embed (ключ активной компании владельца); notesvc хранит эмбеддинги и
-- сам ищет ближайшие. Таблица регенерируемая — в бэкап не входит (денлист).
CREATE TABLE public.note_embeddings (
    note_id    bigint PRIMARY KEY REFERENCES public.notes(id) ON DELETE CASCADE,
    owner_id   bigint NOT NULL,
    embedding  public.vector(1536) NOT NULL,
    model      varchar(64) NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_note_emb_owner ON public.note_embeddings (owner_id);
CREATE INDEX idx_note_emb_hnsw ON public.note_embeddings
    USING hnsw (embedding public.vector_cosine_ops);

-- +goose Down
DROP TABLE IF EXISTS public.note_embeddings;
