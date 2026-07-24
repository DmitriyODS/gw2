-- +goose Up
-- Рассрочка: кредитный счёт-лимит (InstallmentLimit кудосов) на оплату покупок
-- долями. Товар выдаётся сразу, оплата — до InstallmentParts частей в течение
-- недели; пропущенная неделя без платежа наращивает долг (штраф+проценты, как у
-- кредита). Один ряд — одна покупка в рассрочку; активна пока paid < total.
CREATE TABLE pet_installments (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id BIGINT      NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    category   TEXT        NOT NULL, -- shop | house | food | badge | app_theme | house_theme
    item_key   TEXT        NOT NULL,
    item_title TEXT        NOT NULL,
    total      INT         NOT NULL, -- полная сумма к оплате (растёт при просрочке)
    paid       INT         NOT NULL DEFAULT 0,
    parts      INT         NOT NULL, -- задумано долей
    due_at     TIMESTAMPTZ NOT NULL, -- следующий недельный чекпоинт платежа
    penalized  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_pet_installments_user ON pet_installments(user_id);

-- +goose Down
DROP TABLE IF EXISTS pet_installments;
