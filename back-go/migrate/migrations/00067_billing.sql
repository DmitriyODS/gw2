-- +goose Up
-- Подписки, магазин и платежи платформы (billingsvc).
--
-- Тариф описывает ЛИМИТЫ (они конечны и живут в коде — domain.PlanLimits) и
-- ЦЕНЫ (их правит супер-админ — они здесь). Подписка принадлежит пользователю;
-- компания наследует лимиты СВОЕГО СОЗДАТЕЛЯ: платит владелец — его компания и
-- получает участников, портал, статистику, календари и реестры.
--
-- Все суммы — в КОПЕЙКАХ (bigint): рубли с копейками в float хранить нельзя.

-- Тарифы: набор фиксирован кодом (junior/middle/senior), меняются цены,
-- названия и подводки — витрина магазина берёт их отсюда.
CREATE TABLE public.billing_plans (
    code            varchar(32) PRIMARY KEY,
    name            varchar(64) NOT NULL,
    tagline         text NOT NULL DEFAULT '',
    price_month     bigint NOT NULL DEFAULT 0,  -- копейки за месяц
    price_year      bigint NOT NULL DEFAULT 0,  -- копейки за год (полная сумма)
    sort            integer NOT NULL DEFAULT 0,
    is_active       boolean NOT NULL DEFAULT true,
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- Аддоны — докупка сверх тарифа. amount читается по kind: storage — байты,
-- tokens — токены доступа, company/member — штуки.
CREATE TABLE public.billing_addons (
    code            varchar(32) PRIMARY KEY,
    kind            varchar(16) NOT NULL,       -- storage | tokens | company | member
    name            varchar(64) NOT NULL,
    description     text NOT NULL DEFAULT '',
    amount          bigint NOT NULL,
    price_month     bigint NOT NULL DEFAULT 0,
    price_year      bigint NOT NULL DEFAULT 0,  -- 0 — годовой оплаты нет
    -- recurring=false — разовая покупка (пачка токенов не сгорает и не продлевается).
    recurring       boolean NOT NULL DEFAULT true,
    sort            integer NOT NULL DEFAULT 0,
    is_active       boolean NOT NULL DEFAULT true,
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- Подписка пользователя. Строки нет — значит бесплатный «Джун»: не заводим
-- запись каждому зарегистрировавшемуся.
CREATE TABLE public.billing_subscriptions (
    user_id         integer PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    plan_code       varchar(32) NOT NULL REFERENCES public.billing_plans(code),
    period          varchar(8) NOT NULL DEFAULT 'month',   -- month | year
    -- source: purchase — оплачена, grant — выдал супер-админ, grace —
    -- переходный период действующим аккаунтам при выкате биллинга.
    source          varchar(16) NOT NULL DEFAULT 'purchase',
    started_at      timestamptz NOT NULL DEFAULT now(),
    expires_at      timestamptz,                            -- NULL — бессрочно
    auto_renew      boolean NOT NULL DEFAULT true,
    cancelled_at    timestamptz,
    note            text NOT NULL DEFAULT '',
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX billing_subscriptions_expires_idx ON public.billing_subscriptions (expires_at)
    WHERE expires_at IS NOT NULL;

-- Купленные аддоны: складываются к лимитам тарифа. company_id значим только
-- для kind='member' (место сотрудника покупается в конкретную компанию).
CREATE TABLE public.billing_user_addons (
    id              bigserial PRIMARY KEY,
    user_id         integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    addon_code      varchar(32) NOT NULL REFERENCES public.billing_addons(code),
    kind            varchar(16) NOT NULL,
    amount          bigint NOT NULL,
    qty             integer NOT NULL DEFAULT 1,
    company_id      integer REFERENCES public.companies(id) ON DELETE CASCADE,
    period          varchar(8) NOT NULL DEFAULT 'month',
    started_at      timestamptz NOT NULL DEFAULT now(),
    expires_at      timestamptz,                            -- NULL — бессрочно
    auto_renew      boolean NOT NULL DEFAULT true,
    cancelled_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX billing_user_addons_user_idx ON public.billing_user_addons (user_id, kind);

-- Товары магазина: платформенные (author_id IS NULL) и пользовательские —
-- их авторы выкладывают сами и получают выручку за вычетом комиссии.
CREATE TABLE public.billing_products (
    id              bigserial PRIMARY KEY,
    kind            varchar(24) NOT NULL,       -- theme | wallpaper | gradient | pet_skin | pet_decor | other
    title           varchar(120) NOT NULL,
    description     text NOT NULL DEFAULT '',
    price           bigint NOT NULL DEFAULT 0,  -- копейки
    author_id       integer REFERENCES public.users(id) ON DELETE SET NULL,
    -- status: draft — черновик автора, review — на модерации, published —
    -- на витрине, rejected — отклонён, removed — снят с продажи.
    status          varchar(16) NOT NULL DEFAULT 'draft',
    reject_reason   text NOT NULL DEFAULT '',
    cover_path      varchar(500),
    -- payload — сам товар: рецепт темы, обоев, ключ скина питомца.
    payload         jsonb NOT NULL DEFAULT '{}'::jsonb,
    sales_count     integer NOT NULL DEFAULT 0,
    sort            integer NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    published_at    timestamptz
);

CREATE INDEX billing_products_showcase_idx ON public.billing_products (kind, sort, id)
    WHERE status = 'published';
CREATE INDEX billing_products_author_idx ON public.billing_products (author_id, status);

-- Промокоды. kind: percent — скидка в процентах, amount — скидка в копейках,
-- days — бесплатные дни тарифа plan_code, tokens — пачка токенов доступа.
CREATE TABLE public.billing_promos (
    id              bigserial PRIMARY KEY,
    code            varchar(48) NOT NULL,
    kind            varchar(16) NOT NULL,
    value           bigint NOT NULL DEFAULT 0,
    plan_code       varchar(32) REFERENCES public.billing_plans(code),
    -- applies_to: subscription | addon | product | any
    applies_to      varchar(16) NOT NULL DEFAULT 'any',
    max_uses        integer NOT NULL DEFAULT 0,  -- 0 — без ограничения
    per_user_limit  integer NOT NULL DEFAULT 1,
    used_count      integer NOT NULL DEFAULT 0,
    starts_at       timestamptz,
    expires_at      timestamptz,
    is_active       boolean NOT NULL DEFAULT true,
    comment         text NOT NULL DEFAULT '',
    created_by      integer REFERENCES public.users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);

-- Код нечувствителен к регистру: сравниваем и уникальность держим по upper().
CREATE UNIQUE INDEX billing_promos_code_key ON public.billing_promos (upper(code));

-- Заказ — намерение купить. Оплаченный заказ применяется ровно один раз
-- (applied_at), поэтому повторный вебхук платежа ничего не удваивает.
CREATE TABLE public.billing_orders (
    id              bigserial PRIMARY KEY,
    user_id         integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    kind            varchar(16) NOT NULL,        -- subscription | addon | product
    item_code       varchar(32) NOT NULL DEFAULT '',  -- код тарифа или аддона
    product_id      bigint REFERENCES public.billing_products(id) ON DELETE SET NULL,
    period          varchar(8) NOT NULL DEFAULT 'month',
    qty             integer NOT NULL DEFAULT 1,
    company_id      integer REFERENCES public.companies(id) ON DELETE SET NULL,
    amount          bigint NOT NULL DEFAULT 0,   -- к оплате после скидки, копейки
    base_amount     bigint NOT NULL DEFAULT 0,   -- до скидки
    discount        bigint NOT NULL DEFAULT 0,
    promo_id        bigint REFERENCES public.billing_promos(id) ON DELETE SET NULL,
    -- status: pending — ждёт оплаты, paid — оплачен и применён,
    -- canceled | failed | refunded.
    status          varchar(16) NOT NULL DEFAULT 'pending',
    title           varchar(160) NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    paid_at         timestamptz,
    applied_at      timestamptz,
    meta            jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX billing_orders_user_idx ON public.billing_orders (user_id, created_at DESC);
CREATE INDEX billing_orders_status_idx ON public.billing_orders (status, created_at DESC);

-- Списание промокода: один пользователь — ограниченное число активаций.
CREATE TABLE public.billing_promo_redemptions (
    id              bigserial PRIMARY KEY,
    promo_id        bigint NOT NULL REFERENCES public.billing_promos(id) ON DELETE CASCADE,
    user_id         integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    order_id        bigint REFERENCES public.billing_orders(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX billing_promo_redemptions_pair_idx ON public.billing_promo_redemptions (promo_id, user_id);

-- Платёж по заказу. provider='manual' — заглушка до подключения банка: счёт
-- подтверждает супер-админ. Реальный СБП придёт сюда же своим адаптером.
CREATE TABLE public.billing_payments (
    id                  bigserial PRIMARY KEY,
    order_id            bigint NOT NULL REFERENCES public.billing_orders(id) ON DELETE CASCADE,
    provider            varchar(24) NOT NULL DEFAULT 'manual',
    provider_payment_id varchar(120) NOT NULL DEFAULT '',
    amount              bigint NOT NULL DEFAULT 0,
    -- status: pending | succeeded | canceled | failed
    status              varchar(16) NOT NULL DEFAULT 'pending',
    method              varchar(16) NOT NULL DEFAULT 'sbp',
    confirmation_url    text NOT NULL DEFAULT '',
    -- Секрет проверки вебхука конкретного платежа (заглушка живёт без подписи
    -- провайдера: вебхук должен предъявить этот код).
    webhook_secret      varchar(64) NOT NULL DEFAULT '',
    raw                 jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX billing_payments_order_idx ON public.billing_payments (order_id);
CREATE INDEX billing_payments_provider_idx ON public.billing_payments (provider, provider_payment_id);

-- Купленный товар: один и тот же товар не покупается дважды.
CREATE TABLE public.billing_product_purchases (
    id              bigserial PRIMARY KEY,
    product_id      bigint NOT NULL REFERENCES public.billing_products(id) ON DELETE CASCADE,
    user_id         integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    order_id        bigint REFERENCES public.billing_orders(id) ON DELETE SET NULL,
    amount          bigint NOT NULL DEFAULT 0,
    author_share    bigint NOT NULL DEFAULT 0,  -- начислено автору (за вычетом комиссии)
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (product_id, user_id)
);

CREATE INDEX billing_product_purchases_user_idx ON public.billing_product_purchases (user_id, created_at DESC);

-- Кошелёк автора: выручка с продаж копится здесь, выводится заявкой.
CREATE TABLE public.billing_seller_balances (
    user_id         integer PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    balance         bigint NOT NULL DEFAULT 0,
    total_earned    bigint NOT NULL DEFAULT 0,
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.billing_payouts (
    id              bigserial PRIMARY KEY,
    user_id         integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    amount          bigint NOT NULL,
    -- status: requested | paid | rejected
    status          varchar(16) NOT NULL DEFAULT 'requested',
    requisites      text NOT NULL DEFAULT '',
    note            text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    processed_at    timestamptz
);

CREATE INDEX billing_payouts_status_idx ON public.billing_payouts (status, created_at DESC);

-- Баланс токенов доступа к ИИ. Квота тарифа обновляется раз в месяц (лениво,
-- по period_end), докупленные токены не сгорают.
CREATE TABLE public.billing_ai_balances (
    user_id         integer PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    plan_tokens     bigint NOT NULL DEFAULT 0,  -- квота текущего периода
    used_tokens     bigint NOT NULL DEFAULT 0,  -- израсходовано в периоде
    extra_tokens    bigint NOT NULL DEFAULT 0,  -- докупленные, переносятся
    period_start    timestamptz NOT NULL DEFAULT now(),
    period_end      timestamptz NOT NULL DEFAULT now() + interval '1 month',
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- Расход токенов: одна строка на обращение к модели (нужна и пользователю в
-- настройках, и супер-админу в «Аудите платформы»).
CREATE TABLE public.billing_ai_usage (
    id                bigserial PRIMARY KEY,
    user_id           integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    company_id        integer REFERENCES public.companies(id) ON DELETE SET NULL,
    -- Кто фактически работал (компанийные функции тратят токены создателя).
    actor_id          integer REFERENCES public.users(id) ON DELETE SET NULL,
    feature           varchar(24) NOT NULL,     -- assistant | notes | search | tv_fact | support | alice
    model             varchar(64) NOT NULL DEFAULT '',
    prompt_tokens     integer NOT NULL DEFAULT 0,
    completion_tokens integer NOT NULL DEFAULT 0,
    billed_tokens     bigint NOT NULL DEFAULT 0, -- токены доступа (0 — на своём ключе)
    own_key           boolean NOT NULL DEFAULT false,
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX billing_ai_usage_user_idx ON public.billing_ai_usage (user_id, created_at DESC);

-- Занятое место. Считается дельтами: сервис-владелец файла сообщает изменение
-- при заливке и удалении. Разбивка по сервисам — чтобы пользователь видел, чем
-- именно занято хранилище.
CREATE TABLE public.billing_storage_usage (
    user_id         integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    service         varchar(24) NOT NULL,       -- messenger | notes | boards | registry | calendar | portal | avatars
    bytes           bigint NOT NULL DEFAULT 0,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, service)
);

-- Платформенные настройки биллинга (одна строка). Ключ и модели ИИ живут не
-- здесь, а у своего владельца — aisvc.
CREATE TABLE public.billing_settings (
    id                  integer PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    commission_pct      integer NOT NULL DEFAULT 10,   -- удержание платформы с продажи товара автора
    payment_provider    varchar(24) NOT NULL DEFAULT 'manual',
    payment_enabled     boolean NOT NULL DEFAULT false,
    -- Витрину и покупки можно закрыть целиком, не выкатывая релиз.
    store_enabled       boolean NOT NULL DEFAULT true,
    updated_at          timestamptz NOT NULL DEFAULT now()
);

INSERT INTO public.billing_settings (id) VALUES (1);

-- Журнал административных действий: кто, что и над кем сделал.
CREATE TABLE public.platform_audit_log (
    id              bigserial PRIMARY KEY,
    actor_id        integer REFERENCES public.users(id) ON DELETE SET NULL,
    action          varchar(48) NOT NULL,
    target_kind     varchar(24) NOT NULL DEFAULT '',
    target_id       varchar(64) NOT NULL DEFAULT '',
    summary         text NOT NULL DEFAULT '',
    payload         jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX platform_audit_log_recent_idx ON public.platform_audit_log (created_at DESC);
CREATE INDEX platform_audit_log_actor_idx ON public.platform_audit_log (actor_id, created_at DESC);

-- Стартовая линейка тарифов (цены — из «Линейки тарифов», дальше их правит
-- супер-админ в разделе «Аудит платформы»).
INSERT INTO public.billing_plans (code, name, tagline, price_month, price_year, sort) VALUES
    ('junior', 'Джун',   'Бесплатный тариф, чтобы попробовать платформу в деле', 0, 0, 1),
    ('middle', 'Мидл',   'Расширенный тариф для корпоративного использования с доступом к ИИ возможностям', 29900, 238800, 2),
    ('senior', 'Синьор', 'Профессиональный тариф с расширенными возможностями использования', 49900, 478800, 3);

INSERT INTO public.billing_addons (code, kind, name, description, amount, price_month, price_year, recurring, sort) VALUES
    ('storage_5',   'storage', '+5 Гб',           'Расширьте своё хранилище на дополнительные 5 Гб пространства по выгодной цене',   5368709120,  9900,      0, true,  1),
    ('storage_10',  'storage', '+10 Гб',          'Расширьте своё хранилище на дополнительные 10 Гб пространства по выгодной цене', 10737418240, 19900, 178800, true,  2),
    ('storage_20',  'storage', '+20 Гб',          'Расширьте своё хранилище на дополнительные 20 Гб пространства по выгодной цене', 21474836480, 29900, 238800, true,  3),
    ('tokens_1000', 'tokens',  '+1000 токенов',   'Дополнительные токены использования ИИ возможностей для любых целей',            1000,        10000,     0, false, 4),
    ('company_1',   'company', '+1 компания',     'Не хватает компаний? Добавьте ещё одну компанию себе в коллекцию и работайте, как раньше', 1, 20000, 0, true, 5),
    ('member_1',    'member',  '+1 сотрудник',    'Не хватает место в компании? Добавьте место для нового сотрудника в любую свою компанию',  1, 10000, 0, true, 6);

-- Переходный период: все, кто уже пользуется платформой, получают «Синьора» на
-- 30 дней — лимиты не должны обрушиться на действующие компании в день выката.
INSERT INTO public.billing_subscriptions (user_id, plan_code, period, source, expires_at, auto_renew, note)
SELECT u.id, 'senior', 'month', 'grace', now() + interval '30 days', false,
       'Переходный период при запуске подписок'
  FROM public.users u;

-- +goose Down
DROP TABLE IF EXISTS public.platform_audit_log;
DROP TABLE IF EXISTS public.billing_settings;
DROP TABLE IF EXISTS public.billing_storage_usage;
DROP TABLE IF EXISTS public.billing_ai_usage;
DROP TABLE IF EXISTS public.billing_ai_balances;
DROP TABLE IF EXISTS public.billing_payouts;
DROP TABLE IF EXISTS public.billing_seller_balances;
DROP TABLE IF EXISTS public.billing_product_purchases;
DROP TABLE IF EXISTS public.billing_payments;
DROP TABLE IF EXISTS public.billing_promo_redemptions;
DROP TABLE IF EXISTS public.billing_orders;
DROP TABLE IF EXISTS public.billing_promos;
DROP TABLE IF EXISTS public.billing_products;
DROP TABLE IF EXISTS public.billing_user_addons;
DROP TABLE IF EXISTS public.billing_subscriptions;
DROP TABLE IF EXISTS public.billing_addons;
DROP TABLE IF EXISTS public.billing_plans;
