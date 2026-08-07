-- +goose Up
-- ИИ-ассистент подключается к КОНКРЕТНОМУ ПОЛЬЗОВАТЕЛЮ, а не к компании: ключ,
-- модель и тумблер живут рядом с аккаунтом и переезжают за человеком между
-- компаниями (и работают, когда активной компании нет вовсе). Компанийные
-- ai_*-поля companies при этом остаются на месте — на них по-прежнему держатся
-- эмбеддинги задач/заметок и ТВ-факт дня.
CREATE TABLE public.user_ai_settings (
    user_id     integer PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    enabled     boolean NOT NULL DEFAULT true,
    api_key_enc bytea,          -- NULL — ключ не задан, ассистент выключен
    key_hint    text,           -- «sk-…abcd» для карточки в профиле
    model_chat  text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- Диалог ассистента тоже становится личным: компания уходит из ключа, и
-- переключение компании больше не подменяет историю. Прежние компанийные треды
-- одного человека сливаются в один — самый ранний, остальные переезжают в него
-- сообщениями (порядок держит created_at, он у сообщений свой).
ALTER TABLE public.ai_assistant_conversations
    DROP CONSTRAINT IF EXISTS ai_assistant_conversations_user_id_company_id_key;

UPDATE public.ai_assistant_messages m
   SET conversation_id = keep.id
  FROM public.ai_assistant_conversations c
  JOIN LATERAL (
        SELECT k.id
          FROM public.ai_assistant_conversations k
         WHERE k.user_id = c.user_id
         ORDER BY k.created_at, k.id
         LIMIT 1
       ) keep ON true
 WHERE m.conversation_id = c.id AND c.id <> keep.id;

DELETE FROM public.ai_assistant_conversations c
 WHERE c.id <> (
        SELECT k.id
          FROM public.ai_assistant_conversations k
         WHERE k.user_id = c.user_id
         ORDER BY k.created_at, k.id
         LIMIT 1
       );

ALTER TABLE public.ai_assistant_conversations DROP COLUMN company_id;
ALTER TABLE public.ai_assistant_conversations
    ADD CONSTRAINT ai_assistant_conversations_user_id_key UNIQUE (user_id);

-- +goose Down
ALTER TABLE public.ai_assistant_conversations
    DROP CONSTRAINT IF EXISTS ai_assistant_conversations_user_id_key;

ALTER TABLE public.ai_assistant_conversations
    ADD COLUMN company_id integer REFERENCES public.companies(id) ON DELETE CASCADE;

-- Компанию вернуть неоткуда — берём любую компанию владельца; диалоги людей
-- без компании при откате теряются (колонка была NOT NULL).
UPDATE public.ai_assistant_conversations c
   SET company_id = (
        SELECT uc.company_id
          FROM public.user_companies uc
         WHERE uc.user_id = c.user_id
         ORDER BY uc.company_id
         LIMIT 1
       );

DELETE FROM public.ai_assistant_conversations WHERE company_id IS NULL;

ALTER TABLE public.ai_assistant_conversations ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE public.ai_assistant_conversations
    ADD CONSTRAINT ai_assistant_conversations_user_id_company_id_key UNIQUE (user_id, company_id);

DROP TABLE public.user_ai_settings;
