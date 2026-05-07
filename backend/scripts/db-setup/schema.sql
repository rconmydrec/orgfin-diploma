--
-- PostgreSQL database dump
--


-- Dumped from database version 16.3 (Debian 16.3-1.pgdg110+1)
-- Dumped by pg_dump version 18.3

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

-- *not* creating schema, since initdb creates it


--
-- Name: billingperiodcode; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.billingperiodcode AS ENUM (
    'MONTHLY',
    'YEARLY'
);


--
-- Name: paymentprovidertype; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.paymentprovidertype AS ENUM (
    'STRIPE'
);


--
-- Name: periodenum; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.periodenum AS ENUM (
    'DAILY',
    'WEEKLY',
    'MONTHLY',
    'YEARLY',
    'CUSTOM'
);


--
-- Name: plantype; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.plantype AS ENUM (
    'FREE',
    'TRIAL',
    'PREMIUM'
);


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: account_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account_types (
    id integer NOT NULL,
    type_name character varying(100) NOT NULL,
    is_credit boolean DEFAULT false NOT NULL,
    is_deleted boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: account_types_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.account_types_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: account_types_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.account_types_id_seq OWNED BY public.account_types.id;


--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.accounts (
    id integer NOT NULL,
    user_id integer NOT NULL,
    account_type_id integer NOT NULL,
    currency_id integer NOT NULL,
    initial_balance numeric NOT NULL,
    balance numeric NOT NULL,
    name character varying(100) NOT NULL,
    opening_date timestamp with time zone,
    comment character varying,
    is_hidden boolean NOT NULL,
    is_deleted boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    show_in_reports boolean,
    credit_limit numeric,
    is_archived boolean DEFAULT false,
    archived_at timestamp with time zone,
    is_active boolean DEFAULT true NOT NULL
);


--
-- Name: accounts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.accounts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: accounts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.accounts_id_seq OWNED BY public.accounts.id;


--
-- Name: activation_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.activation_tokens (
    id integer NOT NULL,
    user_id integer NOT NULL,
    token character varying(32) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: activation_tokens_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.activation_tokens_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: activation_tokens_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.activation_tokens_id_seq OWNED BY public.activation_tokens.id;


--
-- Name: alembic_version; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alembic_version (
    version_num character varying(32) NOT NULL
);


--
-- Name: billing_periods; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_periods (
    id integer NOT NULL,
    code public.billingperiodcode NOT NULL,
    name character varying(100) NOT NULL,
    duration_days integer NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: billing_periods_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.billing_periods_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: billing_periods_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.billing_periods_id_seq OWNED BY public.billing_periods.id;


--
-- Name: budgets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.budgets (
    id integer NOT NULL,
    user_id integer NOT NULL,
    name character varying(200) NOT NULL,
    target_amount numeric NOT NULL,
    collected_amount numeric NOT NULL,
    period public.periodenum NOT NULL,
    repeat boolean NOT NULL,
    start_date timestamp with time zone NOT NULL,
    end_date timestamp with time zone NOT NULL,
    included_categories text,
    comment character varying,
    is_deleted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    is_archived boolean DEFAULT false NOT NULL,
    currency_id integer NOT NULL,
    is_active boolean DEFAULT true NOT NULL
);


--
-- Name: budgets_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.budgets_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: budgets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.budgets_id_seq OWNED BY public.budgets.id;


--
-- Name: currencies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.currencies (
    id integer NOT NULL,
    code character varying(3) NOT NULL,
    name character varying NOT NULL,
    is_deleted boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: currencies_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.currencies_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: currencies_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.currencies_id_seq OWNED BY public.currencies.id;


--
-- Name: default_categories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.default_categories (
    id integer NOT NULL,
    name character varying NOT NULL,
    parent_id integer,
    is_income boolean DEFAULT false NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: default_categories_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.default_categories_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: default_categories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.default_categories_id_seq OWNED BY public.default_categories.id;


--
-- Name: exchange_rates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.exchange_rates (
    id integer NOT NULL,
    rates jsonb,
    actual_date date NOT NULL,
    base_currency_code character varying(3) NOT NULL,
    service_name character varying(50) NOT NULL,
    is_deleted boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: exchange_rates_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.exchange_rates_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: exchange_rates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.exchange_rates_id_seq OWNED BY public.exchange_rates.id;


--
-- Name: goose_db_version; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.goose_db_version (
    id bigint NOT NULL,
    version_id bigint NOT NULL,
    is_applied boolean NOT NULL,
    tstamp timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: goose_db_version_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.goose_db_version_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: goose_db_version_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.goose_db_version_id_seq OWNED BY public.goose_db_version.id;


--
-- Name: languages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.languages (
    id integer NOT NULL,
    name character varying NOT NULL,
    code character varying(50) NOT NULL,
    is_deleted boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: languages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.languages_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: languages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.languages_id_seq OWNED BY public.languages.id;


--
-- Name: payment_provider_subscriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.payment_provider_subscriptions (
    id integer NOT NULL,
    subscription_id integer NOT NULL,
    provider_type public.paymentprovidertype NOT NULL,
    external_customer_id character varying(100),
    external_subscription_id character varying(100),
    external_schedule_id character varying(100),
    payment_method_id character varying(100),
    last_payment_failed boolean DEFAULT false NOT NULL,
    provider_metadata jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: payment_provider_subscriptions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.payment_provider_subscriptions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: payment_provider_subscriptions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.payment_provider_subscriptions_id_seq OWNED BY public.payment_provider_subscriptions.id;


--
-- Name: plan_prices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan_prices (
    id integer NOT NULL,
    plan_id integer NOT NULL,
    provider_type public.paymentprovidertype NOT NULL,
    external_price_id character varying(100) NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: plan_prices_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.plan_prices_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: plan_prices_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.plan_prices_id_seq OWNED BY public.plan_prices.id;


--
-- Name: planned_transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.planned_transactions (
    id integer NOT NULL,
    user_id integer NOT NULL,
    amount numeric(15,2) NOT NULL,
    label character varying(50),
    notes text,
    is_income boolean NOT NULL,
    planned_date timestamp with time zone NOT NULL,
    is_recurring boolean DEFAULT false NOT NULL,
    recurrence_rule json,
    is_executed boolean DEFAULT false NOT NULL,
    executed_transaction_id integer,
    execution_date timestamp with time zone,
    is_active boolean DEFAULT true NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    currency_id integer NOT NULL
);


--
-- Name: planned_transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.planned_transactions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: planned_transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.planned_transactions_id_seq OWNED BY public.planned_transactions.id;


--
-- Name: subscription_plans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.subscription_plans (
    id integer NOT NULL,
    name character varying(100) NOT NULL,
    plan_type public.plantype NOT NULL,
    billing_period_id integer,
    currency_id integer NOT NULL,
    price numeric(10,2) NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    is_featured boolean DEFAULT false NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    description character varying,
    is_deleted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    translation_key character varying(100)
);


--
-- Name: subscription_plans_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.subscription_plans_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: subscription_plans_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.subscription_plans_id_seq OWNED BY public.subscription_plans.id;


--
-- Name: subscriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.subscriptions (
    id integer NOT NULL,
    user_id integer NOT NULL,
    trial_started_at timestamp with time zone,
    trial_ends_at timestamp with time zone,
    is_active boolean DEFAULT true NOT NULL,
    subscribed_at timestamp with time zone,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    plan_id integer NOT NULL,
    auto_renew boolean DEFAULT true NOT NULL,
    canceled_at timestamp with time zone,
    current_billing_period_id integer,
    pending_downgrade_account_ids jsonb,
    pending_downgrade_budget_id integer,
    pending_plan_id integer
);


--
-- Name: subscriptions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.subscriptions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: subscriptions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.subscriptions_id_seq OWNED BY public.subscriptions.id;


--
-- Name: transaction_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.transaction_templates (
    id integer NOT NULL,
    user_id integer NOT NULL,
    label character varying(255),
    category_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: transaction_templates_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.transaction_templates_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: transaction_templates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.transaction_templates_id_seq OWNED BY public.transaction_templates.id;


--
-- Name: transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.transactions (
    id integer NOT NULL,
    user_id integer NOT NULL,
    account_id integer NOT NULL,
    amount numeric NOT NULL,
    new_balance numeric,
    category_id integer,
    label character varying(255),
    is_income boolean NOT NULL,
    is_transfer boolean DEFAULT false NOT NULL,
    linked_transaction_id integer,
    notes character varying,
    date_time timestamp with time zone,
    is_deleted boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    is_adjustment boolean DEFAULT false NOT NULL,
    exclude_from_reports boolean DEFAULT false NOT NULL,
    base_currency_amount numeric
);


--
-- Name: transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.transactions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.transactions_id_seq OWNED BY public.transactions.id;


--
-- Name: user_categories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_categories (
    id integer NOT NULL,
    user_id integer NOT NULL,
    name character varying NOT NULL,
    parent_id integer,
    is_income boolean DEFAULT false NOT NULL,
    is_deleted boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_categories_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_categories_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_categories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_categories_id_seq OWNED BY public.user_categories.id;


--
-- Name: user_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_settings (
    id integer NOT NULL,
    settings json NOT NULL,
    user_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_settings_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_settings_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_settings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_settings_id_seq OWNED BY public.user_settings.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id integer NOT NULL,
    email character varying NOT NULL,
    first_name character varying,
    last_name character varying,
    password_hash character varying NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    base_currency_id integer NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
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
-- Name: account_types id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_types ALTER COLUMN id SET DEFAULT nextval('public.account_types_id_seq'::regclass);


--
-- Name: accounts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts ALTER COLUMN id SET DEFAULT nextval('public.accounts_id_seq'::regclass);


--
-- Name: activation_tokens id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activation_tokens ALTER COLUMN id SET DEFAULT nextval('public.activation_tokens_id_seq'::regclass);


--
-- Name: billing_periods id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_periods ALTER COLUMN id SET DEFAULT nextval('public.billing_periods_id_seq'::regclass);


--
-- Name: budgets id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.budgets ALTER COLUMN id SET DEFAULT nextval('public.budgets_id_seq'::regclass);


--
-- Name: currencies id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.currencies ALTER COLUMN id SET DEFAULT nextval('public.currencies_id_seq'::regclass);


--
-- Name: default_categories id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.default_categories ALTER COLUMN id SET DEFAULT nextval('public.default_categories_id_seq'::regclass);


--
-- Name: exchange_rates id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exchange_rates ALTER COLUMN id SET DEFAULT nextval('public.exchange_rates_id_seq'::regclass);


--
-- Name: goose_db_version id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.goose_db_version ALTER COLUMN id SET DEFAULT nextval('public.goose_db_version_id_seq'::regclass);


--
-- Name: languages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.languages ALTER COLUMN id SET DEFAULT nextval('public.languages_id_seq'::regclass);


--
-- Name: payment_provider_subscriptions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_provider_subscriptions ALTER COLUMN id SET DEFAULT nextval('public.payment_provider_subscriptions_id_seq'::regclass);


--
-- Name: plan_prices id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_prices ALTER COLUMN id SET DEFAULT nextval('public.plan_prices_id_seq'::regclass);


--
-- Name: planned_transactions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.planned_transactions ALTER COLUMN id SET DEFAULT nextval('public.planned_transactions_id_seq'::regclass);


--
-- Name: subscription_plans id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscription_plans ALTER COLUMN id SET DEFAULT nextval('public.subscription_plans_id_seq'::regclass);


--
-- Name: subscriptions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions ALTER COLUMN id SET DEFAULT nextval('public.subscriptions_id_seq'::regclass);


--
-- Name: transaction_templates id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transaction_templates ALTER COLUMN id SET DEFAULT nextval('public.transaction_templates_id_seq'::regclass);


--
-- Name: transactions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions ALTER COLUMN id SET DEFAULT nextval('public.transactions_id_seq'::regclass);


--
-- Name: user_categories id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_categories ALTER COLUMN id SET DEFAULT nextval('public.user_categories_id_seq'::regclass);


--
-- Name: user_settings id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_settings ALTER COLUMN id SET DEFAULT nextval('public.user_settings_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: account_types account_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_types
    ADD CONSTRAINT account_types_pkey PRIMARY KEY (id);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);


--
-- Name: activation_tokens activation_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activation_tokens
    ADD CONSTRAINT activation_tokens_pkey PRIMARY KEY (id);


--
-- Name: activation_tokens activation_tokens_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activation_tokens
    ADD CONSTRAINT activation_tokens_token_key UNIQUE (token);


--
-- Name: alembic_version alembic_version_pkc; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alembic_version
    ADD CONSTRAINT alembic_version_pkc PRIMARY KEY (version_num);


--
-- Name: billing_periods billing_periods_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_periods
    ADD CONSTRAINT billing_periods_code_key UNIQUE (code);


--
-- Name: billing_periods billing_periods_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.billing_periods
    ADD CONSTRAINT billing_periods_pkey PRIMARY KEY (id);


--
-- Name: budgets budgets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_pkey PRIMARY KEY (id);


--
-- Name: currencies currencies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.currencies
    ADD CONSTRAINT currencies_pkey PRIMARY KEY (id);


--
-- Name: default_categories default_categories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.default_categories
    ADD CONSTRAINT default_categories_pkey PRIMARY KEY (id);


--
-- Name: exchange_rates exchange_rates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exchange_rates
    ADD CONSTRAINT exchange_rates_pkey PRIMARY KEY (id);


--
-- Name: goose_db_version goose_db_version_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.goose_db_version
    ADD CONSTRAINT goose_db_version_pkey PRIMARY KEY (id);


--
-- Name: languages languages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.languages
    ADD CONSTRAINT languages_pkey PRIMARY KEY (id);


--
-- Name: payment_provider_subscriptions payment_provider_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_provider_subscriptions
    ADD CONSTRAINT payment_provider_subscriptions_pkey PRIMARY KEY (id);


--
-- Name: payment_provider_subscriptions payment_provider_subscriptions_subscription_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_provider_subscriptions
    ADD CONSTRAINT payment_provider_subscriptions_subscription_id_key UNIQUE (subscription_id);


--
-- Name: plan_prices plan_prices_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_prices
    ADD CONSTRAINT plan_prices_pkey PRIMARY KEY (id);


--
-- Name: planned_transactions planned_transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.planned_transactions
    ADD CONSTRAINT planned_transactions_pkey PRIMARY KEY (id);


--
-- Name: subscription_plans subscription_plans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscription_plans
    ADD CONSTRAINT subscription_plans_pkey PRIMARY KEY (id);


--
-- Name: subscriptions subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_pkey PRIMARY KEY (id);


--
-- Name: subscriptions subscriptions_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_user_id_key UNIQUE (user_id);


--
-- Name: transaction_templates transaction_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transaction_templates
    ADD CONSTRAINT transaction_templates_pkey PRIMARY KEY (id);


--
-- Name: transactions transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (id);


--
-- Name: exchange_rates unique_service_date; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exchange_rates
    ADD CONSTRAINT unique_service_date UNIQUE (service_name, actual_date);


--
-- Name: plan_prices uq_plan_provider; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_prices
    ADD CONSTRAINT uq_plan_provider UNIQUE (plan_id, provider_type);


--
-- Name: user_categories user_categories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_categories
    ADD CONSTRAINT user_categories_pkey PRIMARY KEY (id);


--
-- Name: user_settings user_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_settings
    ADD CONSTRAINT user_settings_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: ix_account_types_type_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_account_types_type_name ON public.account_types USING btree (type_name);


--
-- Name: ix_accounts_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_accounts_name ON public.accounts USING btree (name);


--
-- Name: ix_accounts_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_accounts_user_id ON public.accounts USING btree (user_id);


--
-- Name: ix_billing_periods_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_billing_periods_code ON public.billing_periods USING btree (code);


--
-- Name: ix_budgets_currency_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_budgets_currency_id ON public.budgets USING btree (currency_id);


--
-- Name: ix_budgets_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_budgets_name ON public.budgets USING btree (name);


--
-- Name: ix_budgets_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_budgets_user_id ON public.budgets USING btree (user_id);


--
-- Name: ix_currencies_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_currencies_code ON public.currencies USING btree (code);


--
-- Name: ix_currencies_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_currencies_name ON public.currencies USING btree (name);


--
-- Name: ix_default_categories_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_default_categories_name ON public.default_categories USING btree (name);


--
-- Name: ix_exchange_rates_actual_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_exchange_rates_actual_date ON public.exchange_rates USING btree (actual_date);


--
-- Name: ix_payment_provider_subscriptions_provider_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_payment_provider_subscriptions_provider_type ON public.payment_provider_subscriptions USING btree (provider_type);


--
-- Name: ix_plan_prices_plan_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_plan_prices_plan_id ON public.plan_prices USING btree (plan_id);


--
-- Name: ix_plan_prices_provider_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_plan_prices_provider_type ON public.plan_prices USING btree (provider_type);


--
-- Name: ix_planned_transactions_executed_transaction_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_planned_transactions_executed_transaction_id ON public.planned_transactions USING btree (executed_transaction_id);


--
-- Name: ix_planned_transactions_label; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_planned_transactions_label ON public.planned_transactions USING btree (label);


--
-- Name: ix_planned_transactions_planned_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_planned_transactions_planned_date ON public.planned_transactions USING btree (planned_date);


--
-- Name: ix_planned_transactions_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_planned_transactions_user_id ON public.planned_transactions USING btree (user_id);


--
-- Name: ix_subscription_plans_plan_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_subscription_plans_plan_type ON public.subscription_plans USING btree (plan_type);


--
-- Name: ix_subscriptions_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ix_subscriptions_user_id ON public.subscriptions USING btree (user_id);


--
-- Name: ix_transaction_templates_category_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_transaction_templates_category_id ON public.transaction_templates USING btree (category_id);


--
-- Name: ix_transaction_templates_label; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_transaction_templates_label ON public.transaction_templates USING btree (label);


--
-- Name: ix_transaction_templates_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_transaction_templates_user_id ON public.transaction_templates USING btree (user_id);


--
-- Name: ix_transactions_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_transactions_account_id ON public.transactions USING btree (account_id);


--
-- Name: ix_transactions_category_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_transactions_category_id ON public.transactions USING btree (category_id);


--
-- Name: ix_transactions_date_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_transactions_date_time ON public.transactions USING btree (date_time);


--
-- Name: ix_transactions_label; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_transactions_label ON public.transactions USING btree (label);


--
-- Name: ix_transactions_linked_transaction_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_transactions_linked_transaction_id ON public.transactions USING btree (linked_transaction_id);


--
-- Name: ix_transactions_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_transactions_user_id ON public.transactions USING btree (user_id);


--
-- Name: ix_user_categories_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_user_categories_name ON public.user_categories USING btree (name);


--
-- Name: ix_user_categories_parent_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_user_categories_parent_id ON public.user_categories USING btree (parent_id);


--
-- Name: ix_user_categories_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_user_categories_user_id ON public.user_categories USING btree (user_id);


--
-- Name: ix_users_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ix_users_email ON public.users USING btree (email);


--
-- Name: ix_users_first_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_users_first_name ON public.users USING btree (first_name);


--
-- Name: ix_users_last_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_users_last_name ON public.users USING btree (last_name);


--
-- Name: accounts accounts_account_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_account_type_id_fkey FOREIGN KEY (account_type_id) REFERENCES public.account_types(id) ON DELETE CASCADE;


--
-- Name: accounts accounts_currency_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES public.currencies(id) ON DELETE CASCADE;


--
-- Name: accounts accounts_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: activation_tokens activation_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activation_tokens
    ADD CONSTRAINT activation_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: budgets budgets_currency_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES public.currencies(id) ON DELETE CASCADE;


--
-- Name: budgets budgets_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: default_categories default_categories_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.default_categories
    ADD CONSTRAINT default_categories_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.default_categories(id) ON DELETE CASCADE;


--
-- Name: subscriptions fk_subscriptions_pending_plan_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT fk_subscriptions_pending_plan_id FOREIGN KEY (pending_plan_id) REFERENCES public.subscription_plans(id);


--
-- Name: payment_provider_subscriptions payment_provider_subscriptions_subscription_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_provider_subscriptions
    ADD CONSTRAINT payment_provider_subscriptions_subscription_id_fkey FOREIGN KEY (subscription_id) REFERENCES public.subscriptions(id) ON DELETE CASCADE;


--
-- Name: plan_prices plan_prices_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_prices
    ADD CONSTRAINT plan_prices_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.subscription_plans(id) ON DELETE CASCADE;


--
-- Name: planned_transactions planned_transactions_currency_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.planned_transactions
    ADD CONSTRAINT planned_transactions_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES public.currencies(id) ON DELETE CASCADE;


--
-- Name: planned_transactions planned_transactions_executed_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.planned_transactions
    ADD CONSTRAINT planned_transactions_executed_transaction_id_fkey FOREIGN KEY (executed_transaction_id) REFERENCES public.transactions(id) ON DELETE SET NULL;


--
-- Name: planned_transactions planned_transactions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.planned_transactions
    ADD CONSTRAINT planned_transactions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: subscription_plans subscription_plans_billing_period_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscription_plans
    ADD CONSTRAINT subscription_plans_billing_period_id_fkey FOREIGN KEY (billing_period_id) REFERENCES public.billing_periods(id) ON DELETE CASCADE;


--
-- Name: subscription_plans subscription_plans_currency_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscription_plans
    ADD CONSTRAINT subscription_plans_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES public.currencies(id) ON DELETE CASCADE;


--
-- Name: subscriptions subscriptions_current_billing_period_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_current_billing_period_id_fkey FOREIGN KEY (current_billing_period_id) REFERENCES public.billing_periods(id);


--
-- Name: subscriptions subscriptions_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.subscription_plans(id);


--
-- Name: subscriptions subscriptions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: transaction_templates transaction_templates_category_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transaction_templates
    ADD CONSTRAINT transaction_templates_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.user_categories(id) ON DELETE CASCADE;


--
-- Name: transaction_templates transaction_templates_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transaction_templates
    ADD CONSTRAINT transaction_templates_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: transactions transactions_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE;


--
-- Name: transactions transactions_category_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.user_categories(id) ON DELETE CASCADE;


--
-- Name: transactions transactions_linked_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_linked_transaction_id_fkey FOREIGN KEY (linked_transaction_id) REFERENCES public.transactions(id) ON DELETE CASCADE;


--
-- Name: transactions transactions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_categories user_categories_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_categories
    ADD CONSTRAINT user_categories_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.user_categories(id) ON DELETE CASCADE;


--
-- Name: user_categories user_categories_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_categories
    ADD CONSTRAINT user_categories_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_settings user_settings_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_settings
    ADD CONSTRAINT user_settings_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: users users_base_currency_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_base_currency_id_fkey FOREIGN KEY (base_currency_id) REFERENCES public.currencies(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--


