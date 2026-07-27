-- +goose Up
-- Режим «в отпуске» переезжает из users в связку user_companies: глобального
-- отпуска «сразу по всем компаниям» у пользователя нет — человек может
-- отдыхать в одной компании и продолжать работать в другой. Ставит и снимает
-- отметку создатель компании (или супер-админ) в карточке сотрудника.
ALTER TABLE public.user_companies
    ADD COLUMN on_vacation boolean NOT NULL DEFAULT false;

-- Перенос существующих отпусков: бывший глобальный флаг раскладывается по всем
-- компаниям пользователя (иначе люди молча вышли бы из отпуска).
UPDATE public.user_companies uc
   SET on_vacation = true
  FROM public.users u
 WHERE u.id = uc.user_id AND u.on_vacation;

ALTER TABLE public.users DROP COLUMN on_vacation;

-- +goose Down
ALTER TABLE public.users
    ADD COLUMN on_vacation boolean NOT NULL DEFAULT false;

UPDATE public.users u
   SET on_vacation = true
  FROM public.user_companies uc
 WHERE uc.user_id = u.id AND uc.on_vacation;

ALTER TABLE public.user_companies DROP COLUMN on_vacation;
