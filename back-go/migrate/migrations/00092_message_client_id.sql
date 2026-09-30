-- +goose Up
-- Ключ идемпотентности отправки: клиент ставит сообщение в очередь без сети и
-- повторяет отправку, пока не получит ответ. Ответ мог потеряться уже после
-- записи (таймаут, обрыв) — повтор с тем же ключом возвращает созданное
-- сообщение, а не второе такое же.
ALTER TABLE public.messages ADD COLUMN client_id text
    CHECK (client_id IS NULL OR length(client_id) BETWEEN 8 AND 64);
CREATE UNIQUE INDEX messages_sender_client_uidx
    ON public.messages (sender_id, client_id) WHERE client_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS public.messages_sender_client_uidx;
ALTER TABLE public.messages DROP COLUMN IF EXISTS client_id;
