-- +goose Up
-- Момент ответа на звонок: длительность разговора считается от него, а не от
-- первого гудка. Прежние состоявшиеся звонки получают started_at — момента
-- ответа у них не записано, а длительность в истории не должна пропасть.
ALTER TABLE public.calls ADD COLUMN answered_at timestamptz;
UPDATE public.calls SET answered_at = started_at WHERE status IN ('active', 'ended');

-- +goose Down
ALTER TABLE public.calls DROP COLUMN IF EXISTS answered_at;
