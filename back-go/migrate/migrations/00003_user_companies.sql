-- +goose Up
-- Членство пользователя в компаниях с ролью в каждой: один человек (один
-- аккаунт) может состоять в нескольких компаниях, роль — своя в каждой.
-- Источник истины для «в каких компаниях состоит юзер и кто он в каждой».
-- users.company_id/role_id остаются «первичной» компанией и ролью в ней
-- (NULL company_id ⇔ Администратор системы); их поддерживает authsvc.
CREATE TABLE public.user_companies (
    user_id    integer NOT NULL,
    company_id integer NOT NULL,
    role_id    integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_companies_pkey PRIMARY KEY (user_id, company_id)
);

ALTER TABLE ONLY public.user_companies
    ADD CONSTRAINT user_companies_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.user_companies
    ADD CONSTRAINT user_companies_company_id_fkey FOREIGN KEY (company_id)
    REFERENCES public.companies(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.user_companies
    ADD CONSTRAINT user_companies_role_id_fkey FOREIGN KEY (role_id)
    REFERENCES public.roles(id);

CREATE INDEX idx_user_companies_company ON public.user_companies USING btree (company_id);

-- Бэкфилл: по одной связке из текущих users.(company_id, role_id) для всех,
-- у кого есть компания. Администраторы системы (company_id IS NULL) членств
-- не получают — они кросс-компанийны через ?company_id=.
INSERT INTO public.user_companies (user_id, company_id, role_id, created_at)
SELECT id, company_id, role_id, COALESCE(created_at, now())
FROM public.users
WHERE company_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS public.user_companies;
