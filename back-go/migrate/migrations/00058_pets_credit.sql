-- +goose Up
-- Кредитный движок v2: условия кредита теперь зависят от КРЕДИТНОГО РЕЙТИНГА
-- (credit_score — растёт за своевременные полные возвраты), у кредита появляется
-- срок (loan_due_at) с грейс-периодом, кэшбэк за возврат в срок и разовая пеня
-- за просрочку. loan_principal — тело текущего кредита (для кэшбэка/пени),
-- loan_penalized — пеня уже применена (чтобы не начислить дважды лениво).
ALTER TABLE pets
    ADD COLUMN credit_score   INT         NOT NULL DEFAULT 0,
    ADD COLUMN loan_principal INT         NOT NULL DEFAULT 0,
    ADD COLUMN loan_due_at    TIMESTAMPTZ,
    ADD COLUMN loan_penalized BOOLEAN     NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE pets
    DROP COLUMN IF EXISTS credit_score,
    DROP COLUMN IF EXISTS loan_principal,
    DROP COLUMN IF EXISTS loan_due_at,
    DROP COLUMN IF EXISTS loan_penalized;
