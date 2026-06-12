-- +goose Up
-- Baseline текущей схемы Groove Work — снимок головы прежних Alembic-миграций
-- (back/migrations, head e7f8a9b0c1d2) на момент фазы 5 ликвидации Flask.
-- На существующей БД (есть alembic_version) cmd/migrate помечает эту ревизию
-- применённой без выполнения; свежая БД получает всю схему отсюда.
-- pg_dump --schema-only --no-owner --no-privileges, без alembic_version.

--
-- PostgreSQL database dump
--


-- Dumped from database version 16.14 (Debian 16.14-1.pgdg12+1)
-- Dumped by pg_dump version 16.14 (Debian 16.14-1.pgdg12+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

-- *not* creating schema, since initdb creates it


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS '';


--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: vector; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA public;


--
-- Name: EXTENSION vector; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION vector IS 'vector data type and ivfflat and hnsw access methods';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: call_participants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.call_participants (
    id integer NOT NULL,
    call_id integer NOT NULL,
    user_id integer NOT NULL,
    role character varying(16) DEFAULT 'invitee'::character varying NOT NULL,
    invited_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    joined_at timestamp with time zone,
    left_at timestamp with time zone,
    declined boolean DEFAULT false NOT NULL
);


--
-- Name: call_participants_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.call_participants_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: call_participants_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.call_participants_id_seq OWNED BY public.call_participants.id;


--
-- Name: calls; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.calls (
    id integer NOT NULL,
    initiator_id integer NOT NULL,
    kind character varying(16) DEFAULT 'p2p'::character varying NOT NULL,
    status character varying(16) DEFAULT 'ringing'::character varying NOT NULL,
    media character varying(8) DEFAULT 'video'::character varying NOT NULL,
    started_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    ended_at timestamp with time zone,
    conversation_id integer,
    company_id integer NOT NULL,
    room_name character varying(64),
    share_code character varying(48)
);


--
-- Name: calls_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.calls_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: calls_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.calls_id_seq OWNED BY public.calls.id;


--
-- Name: comments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.comments (
    id integer NOT NULL,
    task_id integer NOT NULL,
    author_id integer NOT NULL,
    text text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: comments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.comments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: comments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.comments_id_seq OWNED BY public.comments.id;


--
-- Name: companies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.companies (
    id integer NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    director_id integer,
    is_active boolean DEFAULT true NOT NULL,
    settings jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    ai_enabled boolean DEFAULT false NOT NULL,
    ai_api_key_enc bytea,
    ai_key_hint character varying(16),
    ai_model_chat character varying(64) DEFAULT 'gpt-4o-mini'::character varying NOT NULL,
    ai_model_embedding character varying(64) DEFAULT 'text-embedding-3-small'::character varying NOT NULL,
    yg_company_id character varying(64),
    yg_company_name character varying(255),
    yg_project_id character varying(64),
    yg_project_title character varying(255),
    yg_board_id character varying(64),
    yg_board_title character varying(255),
    yg_first_column_id character varying(64),
    yg_completed_column_id character varying(64),
    yg_webhook_id character varying(64),
    yg_webhook_secret character varying(64)
);


--
-- Name: companies_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.companies_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: companies_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.companies_id_seq OWNED BY public.companies.id;


--
-- Name: conversations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.conversations (
    id integer NOT NULL,
    user_a_id integer,
    user_b_id integer,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    last_message_at timestamp with time zone,
    hidden_for_a boolean DEFAULT false NOT NULL,
    hidden_for_b boolean DEFAULT false NOT NULL,
    pinned_at_a timestamp with time zone,
    pinned_at_b timestamp with time zone,
    company_id integer NOT NULL,
    is_dev_chat boolean DEFAULT false NOT NULL,
    is_pet_chat boolean DEFAULT false NOT NULL,
    CONSTRAINT ck_conversation_pair_order CHECK ((((is_dev_chat OR is_pet_chat) AND (NOT (is_dev_chat AND is_pet_chat)) AND (user_a_id IS NOT NULL) AND (user_b_id IS NULL)) OR ((NOT is_dev_chat) AND (NOT is_pet_chat) AND (user_a_id IS NOT NULL) AND (user_b_id IS NOT NULL) AND (user_a_id < user_b_id))))
);


--
-- Name: conversations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.conversations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: conversations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.conversations_id_seq OWNED BY public.conversations.id;


--
-- Name: departments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.departments (
    id integer NOT NULL,
    name character varying(255) NOT NULL,
    company_id integer NOT NULL
);


--
-- Name: departments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.departments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: departments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.departments_id_seq OWNED BY public.departments.id;


--
-- Name: favorites; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.favorites (
    task_id integer NOT NULL,
    user_id integer NOT NULL
);


--
-- Name: feed_comments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feed_comments (
    id integer NOT NULL,
    event_id integer NOT NULL,
    author_id integer,
    is_bot boolean DEFAULT false NOT NULL,
    reply_to_id integer,
    text text NOT NULL,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: feed_comments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.feed_comments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: feed_comments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.feed_comments_id_seq OWNED BY public.feed_comments.id;


--
-- Name: feed_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feed_events (
    id integer NOT NULL,
    company_id integer NOT NULL,
    user_id integer,
    kind character varying(32) NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: feed_events_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.feed_events_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: feed_events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.feed_events_id_seq OWNED BY public.feed_events.id;


--
-- Name: feed_reactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feed_reactions (
    id integer NOT NULL,
    event_id integer NOT NULL,
    user_id integer NOT NULL,
    emoji character varying(16) NOT NULL,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: feed_reactions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.feed_reactions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: feed_reactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.feed_reactions_id_seq OWNED BY public.feed_reactions.id;


--
-- Name: groove_raids; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.groove_raids (
    id integer NOT NULL,
    company_id integer NOT NULL,
    week_start date NOT NULL,
    boss character varying(64) NOT NULL,
    target integer NOT NULL,
    reward character varying(32) DEFAULT 'helmet'::character varying NOT NULL,
    defeated_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: groove_raids_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.groove_raids_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: groove_raids_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.groove_raids_id_seq OWNED BY public.groove_raids.id;


--
-- Name: message_attachments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.message_attachments (
    id integer NOT NULL,
    message_id integer,
    uploader_id integer NOT NULL,
    file_path character varying(500) NOT NULL,
    file_name character varying(255) NOT NULL,
    mime_type character varying(120) NOT NULL,
    size_bytes integer NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: message_attachments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.message_attachments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: message_attachments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.message_attachments_id_seq OWNED BY public.message_attachments.id;


--
-- Name: messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.messages (
    id integer NOT NULL,
    conversation_id integer NOT NULL,
    sender_id integer,
    text text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    read_at timestamp with time zone,
    hidden_for_a boolean DEFAULT false NOT NULL,
    hidden_for_b boolean DEFAULT false NOT NULL,
    reply_to_id integer,
    forwarded_from_user_id integer,
    kind character varying(16) DEFAULT 'text'::character varying NOT NULL,
    call_id integer,
    pinned_at timestamp with time zone,
    pinned_by_id integer,
    task_id integer,
    is_bot boolean DEFAULT false NOT NULL
);


--
-- Name: messages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.messages_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: messages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.messages_id_seq OWNED BY public.messages.id;


--
-- Name: pet_strokes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pet_strokes (
    id integer NOT NULL,
    pet_user_id integer NOT NULL,
    user_id integer NOT NULL,
    day date NOT NULL,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: pet_strokes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.pet_strokes_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: pet_strokes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.pet_strokes_id_seq OWNED BY public.pet_strokes.id;


--
-- Name: pets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pets (
    user_id integer NOT NULL,
    company_id integer NOT NULL,
    name character varying(50) NOT NULL,
    species character varying(16) DEFAULT 'egg'::character varying NOT NULL,
    stage integer DEFAULT 0 NOT NULL,
    xp integer DEFAULT 0 NOT NULL,
    beans integer DEFAULT 0 NOT NULL,
    hat character varying(32),
    accessories jsonb DEFAULT '[]'::jsonb NOT NULL,
    feed_streak integer DEFAULT 0 NOT NULL,
    last_fed_date date,
    created_at timestamp with time zone NOT NULL,
    sick_since timestamp with time zone,
    recovery integer DEFAULT 0 NOT NULL,
    personality character varying(24),
    unlocked_species jsonb DEFAULT '[]'::jsonb NOT NULL,
    quest_date date,
    quest_kind character varying(32),
    quest_target integer,
    quest_progress integer DEFAULT 0 NOT NULL,
    quest_claimed boolean DEFAULT false NOT NULL
);


--
-- Name: roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.roles (
    id integer NOT NULL,
    name character varying(100) NOT NULL,
    level smallint NOT NULL
);


--
-- Name: roles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.roles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.roles_id_seq OWNED BY public.roles.id;


--
-- Name: stages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stages (
    id integer NOT NULL,
    company_id integer NOT NULL,
    name character varying(255) NOT NULL,
    color character varying(16) DEFAULT 'blue'::character varying NOT NULL,
    "order" integer DEFAULT 0 NOT NULL
);


--
-- Name: stages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.stages_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: stages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.stages_id_seq OWNED BY public.stages.id;


--
-- Name: task_embeddings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_embeddings (
    task_id integer NOT NULL,
    company_id integer NOT NULL,
    embedding public.vector(1536) NOT NULL,
    model character varying(64) NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tasks (
    id integer NOT NULL,
    name character varying(500) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    author_id integer NOT NULL,
    link_yougile text,
    received_at timestamp with time zone NOT NULL,
    department_id integer NOT NULL,
    deadline timestamp with time zone,
    is_archived boolean NOT NULL,
    archived_at timestamp with time zone,
    color character varying(20),
    company_id integer NOT NULL,
    responsible_user_id integer,
    stage_id integer,
    yougile_task_id character varying(64),
    yougile_project_id character varying(64),
    yougile_board_id character varying(64),
    yougile_column_id character varying(64),
    yougile_synced_at timestamp with time zone,
    yougile_sync_hash character varying(64),
    yougile_id_short character varying(64)
);


--
-- Name: tasks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.tasks_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: tasks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.tasks_id_seq OWNED BY public.tasks.id;


--
-- Name: unit_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.unit_types (
    id integer NOT NULL,
    name character varying(255) NOT NULL,
    company_id integer NOT NULL
);


--
-- Name: unit_types_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.unit_types_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: unit_types_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.unit_types_id_seq OWNED BY public.unit_types.id;


--
-- Name: units; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.units (
    id integer NOT NULL,
    name character varying(500) NOT NULL,
    user_id integer NOT NULL,
    unit_type_id integer NOT NULL,
    task_id integer NOT NULL,
    is_edited boolean NOT NULL,
    datetime_start timestamp with time zone NOT NULL,
    datetime_end timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    company_id integer NOT NULL
);


--
-- Name: units_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.units_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: units_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.units_id_seq OWNED BY public.units.id;


--
-- Name: user_task_colors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_task_colors (
    user_id integer NOT NULL,
    task_id integer NOT NULL,
    color character varying(20) NOT NULL
);


--
-- Name: user_yougile_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_yougile_accounts (
    id integer NOT NULL,
    user_id integer NOT NULL,
    company_id integer NOT NULL,
    yg_company_id character varying(64) NOT NULL,
    yg_user_id character varying(64),
    yg_login character varying(255) NOT NULL,
    key_ciphertext bytea NOT NULL,
    key_fingerprint character varying(8) NOT NULL,
    last_validated_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_yougile_accounts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_yougile_accounts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_yougile_accounts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_yougile_accounts_id_seq OWNED BY public.user_yougile_accounts.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id integer NOT NULL,
    fio character varying(255) NOT NULL,
    login character varying(100) NOT NULL,
    hash_password text NOT NULL,
    post character varying(255),
    role_id integer NOT NULL,
    avatar_path character varying(500),
    is_default_pass boolean NOT NULL,
    is_hidden boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    last_seen_at timestamp with time zone,
    company_id integer,
    phone character varying(20),
    email character varying(255),
    is_root_admin boolean DEFAULT false NOT NULL
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: call_participants id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.call_participants ALTER COLUMN id SET DEFAULT nextval('public.call_participants_id_seq'::regclass);


--
-- Name: calls id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.calls ALTER COLUMN id SET DEFAULT nextval('public.calls_id_seq'::regclass);


--
-- Name: comments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comments ALTER COLUMN id SET DEFAULT nextval('public.comments_id_seq'::regclass);


--
-- Name: companies id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.companies ALTER COLUMN id SET DEFAULT nextval('public.companies_id_seq'::regclass);


--
-- Name: conversations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.conversations ALTER COLUMN id SET DEFAULT nextval('public.conversations_id_seq'::regclass);


--
-- Name: departments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.departments ALTER COLUMN id SET DEFAULT nextval('public.departments_id_seq'::regclass);


--
-- Name: feed_comments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_comments ALTER COLUMN id SET DEFAULT nextval('public.feed_comments_id_seq'::regclass);


--
-- Name: feed_events id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_events ALTER COLUMN id SET DEFAULT nextval('public.feed_events_id_seq'::regclass);


--
-- Name: feed_reactions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_reactions ALTER COLUMN id SET DEFAULT nextval('public.feed_reactions_id_seq'::regclass);


--
-- Name: groove_raids id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.groove_raids ALTER COLUMN id SET DEFAULT nextval('public.groove_raids_id_seq'::regclass);


--
-- Name: message_attachments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_attachments ALTER COLUMN id SET DEFAULT nextval('public.message_attachments_id_seq'::regclass);


--
-- Name: messages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages ALTER COLUMN id SET DEFAULT nextval('public.messages_id_seq'::regclass);


--
-- Name: pet_strokes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pet_strokes ALTER COLUMN id SET DEFAULT nextval('public.pet_strokes_id_seq'::regclass);


--
-- Name: roles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles ALTER COLUMN id SET DEFAULT nextval('public.roles_id_seq'::regclass);


--
-- Name: stages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stages ALTER COLUMN id SET DEFAULT nextval('public.stages_id_seq'::regclass);


--
-- Name: tasks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks ALTER COLUMN id SET DEFAULT nextval('public.tasks_id_seq'::regclass);


--
-- Name: unit_types id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_types ALTER COLUMN id SET DEFAULT nextval('public.unit_types_id_seq'::regclass);


--
-- Name: units id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.units ALTER COLUMN id SET DEFAULT nextval('public.units_id_seq'::regclass);


--
-- Name: user_yougile_accounts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_yougile_accounts ALTER COLUMN id SET DEFAULT nextval('public.user_yougile_accounts_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: call_participants call_participants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.call_participants
    ADD CONSTRAINT call_participants_pkey PRIMARY KEY (id);


--
-- Name: calls calls_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.calls
    ADD CONSTRAINT calls_pkey PRIMARY KEY (id);


--
-- Name: comments comments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comments
    ADD CONSTRAINT comments_pkey PRIMARY KEY (id);


--
-- Name: companies companies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.companies
    ADD CONSTRAINT companies_pkey PRIMARY KEY (id);


--
-- Name: conversations conversations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.conversations
    ADD CONSTRAINT conversations_pkey PRIMARY KEY (id);


--
-- Name: departments departments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.departments
    ADD CONSTRAINT departments_pkey PRIMARY KEY (id);


--
-- Name: favorites favorites_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.favorites
    ADD CONSTRAINT favorites_pkey PRIMARY KEY (task_id, user_id);


--
-- Name: feed_comments feed_comments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_comments
    ADD CONSTRAINT feed_comments_pkey PRIMARY KEY (id);


--
-- Name: feed_events feed_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_events
    ADD CONSTRAINT feed_events_pkey PRIMARY KEY (id);


--
-- Name: feed_reactions feed_reactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_reactions
    ADD CONSTRAINT feed_reactions_pkey PRIMARY KEY (id);


--
-- Name: groove_raids groove_raids_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.groove_raids
    ADD CONSTRAINT groove_raids_pkey PRIMARY KEY (id);


--
-- Name: message_attachments message_attachments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_attachments
    ADD CONSTRAINT message_attachments_pkey PRIMARY KEY (id);


--
-- Name: messages messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT messages_pkey PRIMARY KEY (id);


--
-- Name: pet_strokes pet_strokes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pet_strokes
    ADD CONSTRAINT pet_strokes_pkey PRIMARY KEY (id);


--
-- Name: pets pets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pets
    ADD CONSTRAINT pets_pkey PRIMARY KEY (user_id);


--
-- Name: roles roles_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_name_key UNIQUE (name);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: stages stages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stages
    ADD CONSTRAINT stages_pkey PRIMARY KEY (id);


--
-- Name: task_embeddings task_embeddings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_embeddings
    ADD CONSTRAINT task_embeddings_pkey PRIMARY KEY (task_id);


--
-- Name: tasks tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_pkey PRIMARY KEY (id);


--
-- Name: unit_types unit_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_types
    ADD CONSTRAINT unit_types_pkey PRIMARY KEY (id);


--
-- Name: units units_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.units
    ADD CONSTRAINT units_pkey PRIMARY KEY (id);


--
-- Name: call_participants uq_callpart_pair; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.call_participants
    ADD CONSTRAINT uq_callpart_pair UNIQUE (call_id, user_id);


--
-- Name: calls uq_calls_share_code; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.calls
    ADD CONSTRAINT uq_calls_share_code UNIQUE (share_code);


--
-- Name: companies uq_companies_name; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.companies
    ADD CONSTRAINT uq_companies_name UNIQUE (name);


--
-- Name: departments uq_departments_company_name; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.departments
    ADD CONSTRAINT uq_departments_company_name UNIQUE (company_id, name);


--
-- Name: feed_reactions uq_feed_reaction; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_reactions
    ADD CONSTRAINT uq_feed_reaction UNIQUE (event_id, user_id, emoji);


--
-- Name: pet_strokes uq_pet_stroke_day; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pet_strokes
    ADD CONSTRAINT uq_pet_stroke_day UNIQUE (pet_user_id, user_id, day);


--
-- Name: groove_raids uq_raid_week; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.groove_raids
    ADD CONSTRAINT uq_raid_week UNIQUE (company_id, week_start);


--
-- Name: stages uq_stages_company_name; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stages
    ADD CONSTRAINT uq_stages_company_name UNIQUE (company_id, name);


--
-- Name: unit_types uq_unit_types_company_name; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_types
    ADD CONSTRAINT uq_unit_types_company_name UNIQUE (company_id, name);


--
-- Name: user_yougile_accounts uq_user_yg_account; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_yougile_accounts
    ADD CONSTRAINT uq_user_yg_account UNIQUE (user_id);


--
-- Name: user_task_colors user_task_colors_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_task_colors
    ADD CONSTRAINT user_task_colors_pkey PRIMARY KEY (user_id, task_id);


--
-- Name: user_yougile_accounts user_yougile_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_yougile_accounts
    ADD CONSTRAINT user_yougile_accounts_pkey PRIMARY KEY (id);


--
-- Name: users users_login_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_login_key UNIQUE (login);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_att_message; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_att_message ON public.message_attachments USING btree (message_id);


--
-- Name: idx_call_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_call_company ON public.calls USING btree (company_id);


--
-- Name: idx_call_started; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_call_started ON public.calls USING btree (started_at);


--
-- Name: idx_call_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_call_status ON public.calls USING btree (status);


--
-- Name: idx_callpart_call; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_callpart_call ON public.call_participants USING btree (call_id);


--
-- Name: idx_callpart_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_callpart_user ON public.call_participants USING btree (user_id);


--
-- Name: idx_comments_author; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_comments_author ON public.comments USING btree (author_id);


--
-- Name: idx_comments_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_comments_task ON public.comments USING btree (task_id, created_at);


--
-- Name: idx_companies_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_companies_active ON public.companies USING btree (is_active);


--
-- Name: idx_conv_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_conv_company ON public.conversations USING btree (company_id);


--
-- Name: idx_conv_last_msg; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_conv_last_msg ON public.conversations USING btree (last_message_at);


--
-- Name: idx_conv_pinned_a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_conv_pinned_a ON public.conversations USING btree (pinned_at_a);


--
-- Name: idx_conv_pinned_b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_conv_pinned_b ON public.conversations USING btree (pinned_at_b);


--
-- Name: idx_conv_user_a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_conv_user_a ON public.conversations USING btree (user_a_id);


--
-- Name: idx_conv_user_b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_conv_user_b ON public.conversations USING btree (user_b_id);


--
-- Name: idx_departments_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_departments_company ON public.departments USING btree (company_id);


--
-- Name: idx_feed_comments_event; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feed_comments_event ON public.feed_comments USING btree (event_id, created_at);


--
-- Name: idx_feed_events_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feed_events_company_id ON public.feed_events USING btree (company_id, id);


--
-- Name: idx_feed_events_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feed_events_user ON public.feed_events USING btree (user_id);


--
-- Name: idx_feed_reactions_event; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feed_reactions_event ON public.feed_reactions USING btree (event_id);


--
-- Name: idx_msg_conv_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_msg_conv_created ON public.messages USING btree (conversation_id, created_at);


--
-- Name: idx_msg_conv_unread; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_msg_conv_unread ON public.messages USING btree (conversation_id) WHERE (read_at IS NULL);


--
-- Name: idx_msg_pinned; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_msg_pinned ON public.messages USING btree (conversation_id, pinned_at);


--
-- Name: idx_msg_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_msg_task ON public.messages USING btree (task_id);


--
-- Name: idx_msg_unread_recipient; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_msg_unread_recipient ON public.messages USING btree (conversation_id, read_at);


--
-- Name: idx_pet_strokes_pet_day; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pet_strokes_pet_day ON public.pet_strokes USING btree (pet_user_id, day);


--
-- Name: idx_pets_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pets_company ON public.pets USING btree (company_id);


--
-- Name: idx_stages_company_order; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_stages_company_order ON public.stages USING btree (company_id, "order");


--
-- Name: idx_task_emb_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_emb_company ON public.task_embeddings USING btree (company_id);


--
-- Name: idx_task_emb_hnsw; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_emb_hnsw ON public.task_embeddings USING hnsw (embedding public.vector_cosine_ops);


--
-- Name: idx_tasks_archived; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_archived ON public.tasks USING btree (is_archived);


--
-- Name: idx_tasks_archived_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_archived_at ON public.tasks USING btree (archived_at) WHERE (is_archived = true);


--
-- Name: idx_tasks_author; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_author ON public.tasks USING btree (author_id);


--
-- Name: idx_tasks_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_company ON public.tasks USING btree (company_id);


--
-- Name: idx_tasks_company_active_received; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_company_active_received ON public.tasks USING btree (company_id, received_at) WHERE (is_archived = false);


--
-- Name: idx_tasks_company_author; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_company_author ON public.tasks USING btree (company_id, author_id);


--
-- Name: idx_tasks_company_department; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_company_department ON public.tasks USING btree (company_id, department_id);


--
-- Name: idx_tasks_company_responsible; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_company_responsible ON public.tasks USING btree (company_id, responsible_user_id);


--
-- Name: idx_tasks_company_stage; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_company_stage ON public.tasks USING btree (company_id, stage_id);


--
-- Name: idx_tasks_dept; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_dept ON public.tasks USING btree (department_id);


--
-- Name: idx_tasks_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_name_trgm ON public.tasks USING gin (lower((name)::text) public.gin_trgm_ops);


--
-- Name: idx_tasks_received; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_received ON public.tasks USING btree (received_at);


--
-- Name: idx_tasks_responsible; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_responsible ON public.tasks USING btree (responsible_user_id);


--
-- Name: idx_tasks_stage; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_stage ON public.tasks USING btree (stage_id);


--
-- Name: idx_unit_types_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_unit_types_company ON public.unit_types USING btree (company_id);


--
-- Name: idx_units_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_units_active ON public.units USING btree (user_id) WHERE (datetime_end IS NULL);


--
-- Name: idx_units_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_units_company ON public.units USING btree (company_id);


--
-- Name: idx_units_company_start; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_units_company_start ON public.units USING btree (company_id, datetime_start);


--
-- Name: idx_units_company_user_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_units_company_user_active ON public.units USING btree (company_id, user_id) WHERE (datetime_end IS NULL);


--
-- Name: idx_units_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_units_task ON public.units USING btree (task_id);


--
-- Name: idx_units_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_units_user ON public.units USING btree (user_id);


--
-- Name: idx_user_yg_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_yg_company ON public.user_yougile_accounts USING btree (company_id);


--
-- Name: idx_users_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_company ON public.users USING btree (company_id);


--
-- Name: idx_users_company_visible; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_company_visible ON public.users USING btree (company_id) WHERE (is_hidden = false);


--
-- Name: idx_users_fio_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_fio_trgm ON public.users USING gin (lower((fio)::text) public.gin_trgm_ops);


--
-- Name: idx_users_login; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_login ON public.users USING btree (login);


--
-- Name: idx_users_login_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_login_trgm ON public.users USING gin (lower((login)::text) public.gin_trgm_ops);


--
-- Name: idx_users_role; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_role ON public.users USING btree (role_id);


--
-- Name: idx_users_visible; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_visible ON public.users USING btree (is_hidden) WHERE (is_hidden = false);


--
-- Name: idx_utc_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_utc_task ON public.user_task_colors USING btree (task_id);


--
-- Name: uq_conversation_dev_user; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_conversation_dev_user ON public.conversations USING btree (user_a_id) WHERE (is_dev_chat = true);


--
-- Name: uq_conversation_pair; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_conversation_pair ON public.conversations USING btree (user_a_id, user_b_id) WHERE (is_dev_chat = false);


--
-- Name: uq_tasks_yougile_per_company; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_tasks_yougile_per_company ON public.tasks USING btree (company_id, yougile_task_id) WHERE (yougile_task_id IS NOT NULL);


--
-- Name: uq_users_email_lower; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_users_email_lower ON public.users USING btree (lower((email)::text)) WHERE (email IS NOT NULL);


--
-- Name: call_participants call_participants_call_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.call_participants
    ADD CONSTRAINT call_participants_call_id_fkey FOREIGN KEY (call_id) REFERENCES public.calls(id) ON DELETE CASCADE;


--
-- Name: call_participants call_participants_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.call_participants
    ADD CONSTRAINT call_participants_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: calls calls_conversation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.calls
    ADD CONSTRAINT calls_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES public.conversations(id) ON DELETE SET NULL;


--
-- Name: calls calls_initiator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.calls
    ADD CONSTRAINT calls_initiator_id_fkey FOREIGN KEY (initiator_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: comments comments_author_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comments
    ADD CONSTRAINT comments_author_id_fkey FOREIGN KEY (author_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: comments comments_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comments
    ADD CONSTRAINT comments_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: conversations conversations_user_a_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.conversations
    ADD CONSTRAINT conversations_user_a_id_fkey FOREIGN KEY (user_a_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: conversations conversations_user_b_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.conversations
    ADD CONSTRAINT conversations_user_b_id_fkey FOREIGN KEY (user_b_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: favorites favorites_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.favorites
    ADD CONSTRAINT favorites_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: favorites favorites_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.favorites
    ADD CONSTRAINT favorites_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: feed_comments feed_comments_author_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_comments
    ADD CONSTRAINT feed_comments_author_id_fkey FOREIGN KEY (author_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: feed_comments feed_comments_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_comments
    ADD CONSTRAINT feed_comments_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.feed_events(id) ON DELETE CASCADE;


--
-- Name: feed_comments feed_comments_reply_to_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_comments
    ADD CONSTRAINT feed_comments_reply_to_id_fkey FOREIGN KEY (reply_to_id) REFERENCES public.feed_comments(id) ON DELETE SET NULL;


--
-- Name: feed_events feed_events_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_events
    ADD CONSTRAINT feed_events_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: feed_events feed_events_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_events
    ADD CONSTRAINT feed_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: feed_reactions feed_reactions_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_reactions
    ADD CONSTRAINT feed_reactions_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.feed_events(id) ON DELETE CASCADE;


--
-- Name: feed_reactions feed_reactions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_reactions
    ADD CONSTRAINT feed_reactions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: calls fk_calls_company; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.calls
    ADD CONSTRAINT fk_calls_company FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: companies fk_companies_director; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.companies
    ADD CONSTRAINT fk_companies_director FOREIGN KEY (director_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: conversations fk_conversations_company; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.conversations
    ADD CONSTRAINT fk_conversations_company FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: departments fk_departments_company; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.departments
    ADD CONSTRAINT fk_departments_company FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: messages fk_msg_call; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT fk_msg_call FOREIGN KEY (call_id) REFERENCES public.calls(id) ON DELETE SET NULL;


--
-- Name: messages fk_msg_forwarded_from; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT fk_msg_forwarded_from FOREIGN KEY (forwarded_from_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: messages fk_msg_pinned_by; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT fk_msg_pinned_by FOREIGN KEY (pinned_by_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: messages fk_msg_reply_to; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT fk_msg_reply_to FOREIGN KEY (reply_to_id) REFERENCES public.messages(id) ON DELETE SET NULL;


--
-- Name: messages fk_msg_task_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT fk_msg_task_id FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE SET NULL;


--
-- Name: tasks fk_tasks_company; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT fk_tasks_company FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: tasks fk_tasks_responsible_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT fk_tasks_responsible_user_id FOREIGN KEY (responsible_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: tasks fk_tasks_stage_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT fk_tasks_stage_id FOREIGN KEY (stage_id) REFERENCES public.stages(id) ON DELETE SET NULL;


--
-- Name: unit_types fk_unit_types_company; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_types
    ADD CONSTRAINT fk_unit_types_company FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: units fk_units_company; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.units
    ADD CONSTRAINT fk_units_company FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: users fk_users_company; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT fk_users_company FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE SET NULL;


--
-- Name: groove_raids groove_raids_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.groove_raids
    ADD CONSTRAINT groove_raids_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: message_attachments message_attachments_message_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_attachments
    ADD CONSTRAINT message_attachments_message_id_fkey FOREIGN KEY (message_id) REFERENCES public.messages(id) ON DELETE CASCADE;


--
-- Name: message_attachments message_attachments_uploader_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_attachments
    ADD CONSTRAINT message_attachments_uploader_id_fkey FOREIGN KEY (uploader_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: messages messages_conversation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT messages_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES public.conversations(id) ON DELETE CASCADE;


--
-- Name: messages messages_sender_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT messages_sender_id_fkey FOREIGN KEY (sender_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: pet_strokes pet_strokes_pet_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pet_strokes
    ADD CONSTRAINT pet_strokes_pet_user_id_fkey FOREIGN KEY (pet_user_id) REFERENCES public.pets(user_id) ON DELETE CASCADE;


--
-- Name: pet_strokes pet_strokes_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pet_strokes
    ADD CONSTRAINT pet_strokes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: pets pets_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pets
    ADD CONSTRAINT pets_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: pets pets_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pets
    ADD CONSTRAINT pets_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: stages stages_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stages
    ADD CONSTRAINT stages_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: task_embeddings task_embeddings_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_embeddings
    ADD CONSTRAINT task_embeddings_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: task_embeddings task_embeddings_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_embeddings
    ADD CONSTRAINT task_embeddings_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: tasks tasks_author_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_author_id_fkey FOREIGN KEY (author_id) REFERENCES public.users(id);


--
-- Name: tasks tasks_department_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_department_id_fkey FOREIGN KEY (department_id) REFERENCES public.departments(id);


--
-- Name: units units_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.units
    ADD CONSTRAINT units_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: units units_unit_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.units
    ADD CONSTRAINT units_unit_type_id_fkey FOREIGN KEY (unit_type_id) REFERENCES public.unit_types(id) ON DELETE CASCADE;


--
-- Name: units units_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.units
    ADD CONSTRAINT units_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: user_task_colors user_task_colors_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_task_colors
    ADD CONSTRAINT user_task_colors_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: user_task_colors user_task_colors_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_task_colors
    ADD CONSTRAINT user_task_colors_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_yougile_accounts user_yougile_accounts_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_yougile_accounts
    ADD CONSTRAINT user_yougile_accounts_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: user_yougile_accounts user_yougile_accounts_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_yougile_accounts
    ADD CONSTRAINT user_yougile_accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: users users_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- PostgreSQL database dump complete
--

-- +goose Down
-- Откат baseline не поддерживается: это снимок всей схемы платформы.
SELECT 1;
