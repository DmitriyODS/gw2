-- +goose Up
-- Выдача учётного реестра различает ПОЛУЧАТЕЛЯ и ОТВЕТСТВЕННОГО: вещь уходит в
-- отдел, на объект или в бригаду, а отвечает за неё конкретный человек с
-- телефоном. Раньше это было одно поле, и на вопрос «куда ушло» ответа не было.
ALTER TABLE public.registry_issues
    ADD COLUMN issued_to varchar(200) NOT NULL DEFAULT '';

-- Прежние выдачи: получателем считаем того же, кто записан ответственным, —
-- других сведений о них нет, а пустой получатель в истории выглядел бы потерей.
UPDATE public.registry_issues SET issued_to = holder_name WHERE issued_to = '';

-- +goose Down
ALTER TABLE public.registry_issues DROP COLUMN IF EXISTS issued_to;
