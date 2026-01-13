-- AI Orchestration Database Schema
-- 
-- Usage:
--   make db-apply   - Apply this schema to database
--   make db-reset   - Drop and recreate all tables
--   make db-seed    - Load initial data
--
-- This file is the single source of truth for the database schema.

--
-- PostgreSQL database dump
--

\restrict qS68oUh7R0Ylf9hxaFCN1NsqAxjwYu48PFRRQDDRLom2r3jbe88asrCBhclQEWj

-- Dumped from database version 16.11
-- Dumped by pg_dump version 16.11

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

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: adapters; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.adapters (
    id character varying(100) NOT NULL,
    tenant_id uuid,
    name character varying(255) NOT NULL,
    description text,
    type character varying(50) NOT NULL,
    config jsonb,
    input_schema jsonb,
    output_schema jsonb,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: audit_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_logs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    actor_id uuid,
    actor_email character varying(255),
    action character varying(100) NOT NULL,
    resource_type character varying(100) NOT NULL,
    resource_id uuid,
    metadata jsonb,
    ip_address inet,
    user_agent text,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: block_definitions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.block_definitions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    slug character varying(100) NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    category character varying(50) NOT NULL,
    icon character varying(50),
    config_schema jsonb DEFAULT '{}'::jsonb NOT NULL,
    input_schema jsonb,
    output_schema jsonb,
    error_codes jsonb DEFAULT '[]'::jsonb,
    enabled boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    output_ports jsonb DEFAULT '[]'::jsonb,
    input_ports jsonb DEFAULT '[]'::jsonb,
    required_credentials jsonb DEFAULT '[]'::jsonb,
    is_public boolean DEFAULT false,
    code text,
    ui_config jsonb DEFAULT '{}'::jsonb NOT NULL,
    is_system boolean DEFAULT false NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    CONSTRAINT valid_block_category CHECK (((category)::text = ANY ((ARRAY['ai'::character varying, 'logic'::character varying, 'integration'::character varying, 'data'::character varying, 'control'::character varying, 'utility'::character varying])::text[])))
);


--
-- Name: COLUMN block_definitions.required_credentials; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.block_definitions.required_credentials IS 'JSON array declaring required credentials: [{name, type, scope, description, required}]';


--
-- Name: COLUMN block_definitions.is_public; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.block_definitions.is_public IS 'Whether tenant block is visible to other tenants';


--
-- Name: COLUMN block_definitions.code; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.block_definitions.code IS 'JavaScript code executed in sandbox. All blocks are code-based.';


--
-- Name: COLUMN block_definitions.ui_config; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.block_definitions.ui_config IS 'UI metadata: icon, color, configSchema for workflow editor';


--
-- Name: COLUMN block_definitions.is_system; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.block_definitions.is_system IS 'System blocks can only be edited by admins';


--
-- Name: COLUMN block_definitions.version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.block_definitions.version IS 'Version number, incremented on each update';


--
-- Name: block_group_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.block_group_runs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    run_id uuid NOT NULL,
    block_group_id uuid NOT NULL,
    status character varying(50) DEFAULT 'pending'::character varying,
    iteration integer DEFAULT 0,
    input jsonb,
    output jsonb,
    error text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT valid_block_group_run_status CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'running'::character varying, 'completed'::character varying, 'failed'::character varying, 'skipped'::character varying])::text[])))
);


--
-- Name: block_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.block_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workflow_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    type character varying(50) NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    parent_group_id uuid,
    position_x integer DEFAULT 0,
    position_y integer DEFAULT 0,
    width integer DEFAULT 400,
    height integer DEFAULT 300,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT valid_block_group_type CHECK (((type)::text = ANY ((ARRAY['parallel'::character varying, 'try_catch'::character varying, 'if_else'::character varying, 'switch_case'::character varying, 'foreach'::character varying, 'while'::character varying])::text[])))
);


--
-- Name: TABLE block_groups; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.block_groups IS 'Control flow constructs that group multiple steps';


--
-- Name: COLUMN block_groups.type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.block_groups.type IS 'Type of control flow: parallel, try_catch, if_else, switch_case, foreach, while';


--
-- Name: COLUMN block_groups.config; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.block_groups.config IS 'Type-specific configuration (JSON)';


--
-- Name: COLUMN block_groups.parent_group_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.block_groups.parent_group_id IS 'Reference to parent group for nested structures';


--
-- Name: block_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.block_versions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    block_id uuid NOT NULL,
    version integer NOT NULL,
    code text NOT NULL,
    config_schema jsonb NOT NULL,
    input_schema jsonb,
    output_schema jsonb,
    ui_config jsonb NOT NULL,
    change_summary text,
    changed_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE block_versions; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.block_versions IS 'Version history for block definitions, enables rollback';


--
-- Name: copilot_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.copilot_messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    session_id uuid NOT NULL,
    role character varying(20) NOT NULL,
    content text NOT NULL,
    metadata jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT copilot_messages_role_check CHECK (((role)::text = ANY ((ARRAY['user'::character varying, 'assistant'::character varying])::text[])))
);


--
-- Name: copilot_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.copilot_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    user_id character varying(255) NOT NULL,
    workflow_id uuid NOT NULL,
    title character varying(255),
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: credentials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.credentials (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(200) NOT NULL,
    description text,
    credential_type character varying(50) NOT NULL,
    encrypted_data bytea NOT NULL,
    encrypted_dek bytea NOT NULL,
    data_nonce bytea NOT NULL,
    dek_nonce bytea NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    expires_at timestamp with time zone,
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT valid_credential_status CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'expired'::character varying, 'revoked'::character varying, 'error'::character varying])::text[]))),
    CONSTRAINT valid_credential_type CHECK (((credential_type)::text = ANY ((ARRAY['oauth2'::character varying, 'api_key'::character varying, 'basic'::character varying, 'bearer'::character varying, 'custom'::character varying])::text[])))
);


--
-- Name: TABLE credentials; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.credentials IS 'Stores encrypted API credentials for external service authentication';


--
-- Name: COLUMN credentials.encrypted_data; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.credentials.encrypted_data IS 'AES-256-GCM encrypted credential data (secrets)';


--
-- Name: COLUMN credentials.encrypted_dek; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.credentials.encrypted_dek IS 'Encrypted Data Encryption Key (envelope encryption)';


--
-- Name: COLUMN credentials.data_nonce; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.credentials.data_nonce IS '12-byte nonce/IV for data encryption';


--
-- Name: COLUMN credentials.dek_nonce; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.credentials.dek_nonce IS '12-byte nonce/IV for DEK encryption';


--
-- Name: COLUMN credentials.metadata; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.credentials.metadata IS 'Non-sensitive metadata (e.g., service name, account info)';


--
-- Name: edges; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.edges (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    source_step_id uuid NOT NULL,
    target_step_id uuid NOT NULL,
    condition text,
    created_at timestamp with time zone DEFAULT now(),
    source_port character varying(50) DEFAULT ''::character varying,
    target_port character varying(50) DEFAULT ''::character varying
);


--
-- Name: runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.runs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    workflow_id uuid NOT NULL,
    workflow_version integer NOT NULL,
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    mode character varying(50) DEFAULT 'production'::character varying NOT NULL,
    input jsonb,
    output jsonb,
    error text,
    triggered_by character varying(50) DEFAULT 'manual'::character varying NOT NULL,
    triggered_by_user uuid,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    trigger_source character varying(100),
    trigger_metadata jsonb DEFAULT '{}'::jsonb
);


--
-- Name: COLUMN runs.trigger_source; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.runs.trigger_source IS 'Internal trigger source identifier: copilot, audit-system, etc.';


--
-- Name: COLUMN runs.trigger_metadata; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.runs.trigger_metadata IS 'Additional metadata about the trigger: feature, user_id, session_id, etc.';


--
-- Name: schedules; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schedules (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    workflow_id uuid NOT NULL,
    workflow_version integer DEFAULT 1 NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    cron_expression character varying(100) NOT NULL,
    timezone character varying(50) DEFAULT 'UTC'::character varying NOT NULL,
    input jsonb,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    next_run_at timestamp with time zone,
    last_run_at timestamp with time zone,
    last_run_id uuid,
    run_count integer DEFAULT 0 NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: secrets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.secrets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    encrypted_value text NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: step_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.step_runs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    run_id uuid NOT NULL,
    step_id uuid NOT NULL,
    step_name character varying(255) NOT NULL,
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    attempt integer DEFAULT 1 NOT NULL,
    input jsonb,
    output jsonb,
    error text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    duration_ms integer,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: steps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.steps (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    type character varying(50) NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    position_x integer DEFAULT 0,
    position_y integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    block_group_id uuid,
    group_role character varying(50),
    credential_bindings jsonb DEFAULT '{}'::jsonb,
    block_definition_id uuid
);


--
-- Name: COLUMN steps.block_group_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.steps.block_group_id IS 'Reference to containing block group (NULL if not in a group)';


--
-- Name: COLUMN steps.group_role; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.steps.group_role IS 'Role within block group: body, try, catch, finally, then, else, case_N, default';


--
-- Name: COLUMN steps.credential_bindings; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.steps.credential_bindings IS 'Mapping of credential names to tenant credential IDs';


--
-- Name: COLUMN steps.block_definition_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.steps.block_definition_id IS 'Reference to block_definitions registry';


--
-- Name: system_credentials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_credentials (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(200) NOT NULL,
    description text,
    credential_type character varying(50) NOT NULL,
    encrypted_data bytea NOT NULL,
    encrypted_dek bytea NOT NULL,
    data_nonce bytea NOT NULL,
    dek_nonce bytea NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    expires_at timestamp with time zone,
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT valid_system_credential_status CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'expired'::character varying, 'revoked'::character varying, 'error'::character varying])::text[]))),
    CONSTRAINT valid_system_credential_type CHECK (((credential_type)::text = ANY ((ARRAY['oauth2'::character varying, 'api_key'::character varying, 'basic'::character varying, 'bearer'::character varying, 'custom'::character varying])::text[])))
);


--
-- Name: TABLE system_credentials; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.system_credentials IS 'Operator-managed credentials for system blocks (not accessible by tenants)';


--
-- Name: tenants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenants (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(255) NOT NULL,
    settings jsonb DEFAULT '{}'::jsonb,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    plan character varying(50) DEFAULT 'free'::character varying NOT NULL,
    owner_email character varying(255),
    owner_name character varying(255),
    billing_email character varying(255),
    metadata jsonb DEFAULT '{}'::jsonb,
    feature_flags jsonb DEFAULT '{}'::jsonb,
    limits jsonb DEFAULT '{}'::jsonb,
    suspended_at timestamp with time zone,
    suspended_reason text
);


--
-- Name: COLUMN tenants.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.status IS 'Tenant status: active, suspended, pending, inactive';


--
-- Name: COLUMN tenants.plan; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.plan IS 'Subscription plan: free, starter, professional, enterprise';


--
-- Name: COLUMN tenants.owner_email; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.owner_email IS 'Primary contact email for the tenant';


--
-- Name: COLUMN tenants.owner_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.owner_name IS 'Primary contact name for the tenant';


--
-- Name: COLUMN tenants.billing_email; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.billing_email IS 'Email for billing notifications';


--
-- Name: COLUMN tenants.metadata; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.metadata IS 'Additional tenant metadata (industry, company_size, website, country, notes)';


--
-- Name: COLUMN tenants.feature_flags; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.feature_flags IS 'Feature flags: copilot_enabled, advanced_analytics, custom_blocks, api_access, sso_enabled, audit_logs, max_concurrent_runs';


--
-- Name: COLUMN tenants.limits; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.limits IS 'Resource limits: max_workflows, max_runs_per_day, max_users, max_credentials, max_storage_mb, retention_days';


--
-- Name: COLUMN tenants.suspended_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.suspended_at IS 'Timestamp when tenant was suspended';


--
-- Name: COLUMN tenants.suspended_reason; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.suspended_reason IS 'Reason for tenant suspension';


--
-- Name: usage_budgets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.usage_budgets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    workflow_id uuid,
    budget_type character varying(50) NOT NULL,
    budget_amount_usd numeric(12,2) NOT NULL,
    alert_threshold numeric(3,2) DEFAULT 0.80 NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE usage_budgets; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.usage_budgets IS 'Budget settings for cost control and alerts';


--
-- Name: COLUMN usage_budgets.alert_threshold; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.usage_budgets.alert_threshold IS 'Percentage (0.00-1.00) at which to trigger alert';


--
-- Name: usage_daily_aggregates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.usage_daily_aggregates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    workflow_id uuid,
    date date NOT NULL,
    provider character varying(50) NOT NULL,
    model character varying(100) NOT NULL,
    total_requests integer DEFAULT 0 NOT NULL,
    successful_requests integer DEFAULT 0 NOT NULL,
    failed_requests integer DEFAULT 0 NOT NULL,
    total_input_tokens bigint DEFAULT 0 NOT NULL,
    total_output_tokens bigint DEFAULT 0 NOT NULL,
    total_cost_usd numeric(12,6) DEFAULT 0 NOT NULL,
    avg_latency_ms integer,
    min_latency_ms integer,
    max_latency_ms integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE usage_daily_aggregates; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.usage_daily_aggregates IS 'Pre-aggregated daily usage data for dashboard performance';


--
-- Name: usage_records; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.usage_records (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    workflow_id uuid,
    run_id uuid,
    step_run_id uuid,
    provider character varying(50) NOT NULL,
    model character varying(100) NOT NULL,
    operation character varying(50) NOT NULL,
    input_tokens integer DEFAULT 0 NOT NULL,
    output_tokens integer DEFAULT 0 NOT NULL,
    total_tokens integer DEFAULT 0 NOT NULL,
    input_cost_usd numeric(12,8) DEFAULT 0 NOT NULL,
    output_cost_usd numeric(12,8) DEFAULT 0 NOT NULL,
    total_cost_usd numeric(12,8) DEFAULT 0 NOT NULL,
    latency_ms integer,
    success boolean DEFAULT true NOT NULL,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE usage_records; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.usage_records IS 'Individual LLM API call records with token usage and cost';


--
-- Name: COLUMN usage_records.provider; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.usage_records.provider IS 'LLM provider: openai, anthropic, google, etc.';


--
-- Name: COLUMN usage_records.model; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.usage_records.model IS 'Model identifier: gpt-4o, claude-3-opus, etc.';


--
-- Name: COLUMN usage_records.operation; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.usage_records.operation IS 'Operation type: chat, completion, embedding, etc.';


--
-- Name: COLUMN usage_records.total_cost_usd; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.usage_records.total_cost_usd IS 'Total cost in USD with 8 decimal precision';


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    email character varying(255) NOT NULL,
    name character varying(255),
    role character varying(50) DEFAULT 'viewer'::character varying NOT NULL,
    last_login_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: webhooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhooks (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    workflow_id uuid NOT NULL,
    workflow_version integer DEFAULT 0 NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    secret character varying(255) NOT NULL,
    input_mapping jsonb,
    enabled boolean DEFAULT true NOT NULL,
    last_triggered_at timestamp with time zone,
    trigger_count integer DEFAULT 0 NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: workflow_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_versions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    version integer NOT NULL,
    definition jsonb NOT NULL,
    saved_by uuid,
    saved_at timestamp with time zone DEFAULT now()
);


--
-- Name: workflows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflows (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    status character varying(50) DEFAULT 'draft'::character varying NOT NULL,
    version integer DEFAULT 0 NOT NULL,
    input_schema jsonb,
    output_schema jsonb,
    draft jsonb,
    created_by uuid,
    published_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    is_system boolean DEFAULT false NOT NULL,
    system_slug character varying(100)
);


--
-- Name: TABLE workflows; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.workflows IS 'Workflow definitions. System workflows (is_system=true) are used for internal features like Copilot.';


--
-- Name: COLUMN workflows.is_system; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.workflows.is_system IS 'True for system workflows (e.g., Copilot). These are accessible across all tenants.';


--
-- Name: COLUMN workflows.system_slug; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.workflows.system_slug IS 'Unique slug for system workflows (e.g., copilot-generate). Used for internal lookups.';


--
-- Name: adapters adapters_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.adapters
    ADD CONSTRAINT adapters_pkey PRIMARY KEY (id);


--
-- Name: audit_logs audit_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);


--
-- Name: block_definitions block_definitions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_definitions
    ADD CONSTRAINT block_definitions_pkey PRIMARY KEY (id);


--
-- Name: block_group_runs block_group_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_group_runs
    ADD CONSTRAINT block_group_runs_pkey PRIMARY KEY (id);


--
-- Name: block_groups block_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_groups
    ADD CONSTRAINT block_groups_pkey PRIMARY KEY (id);


--
-- Name: block_versions block_versions_block_id_version_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_versions
    ADD CONSTRAINT block_versions_block_id_version_key UNIQUE (block_id, version);


--
-- Name: block_versions block_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_versions
    ADD CONSTRAINT block_versions_pkey PRIMARY KEY (id);


--
-- Name: copilot_messages copilot_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.copilot_messages
    ADD CONSTRAINT copilot_messages_pkey PRIMARY KEY (id);


--
-- Name: copilot_sessions copilot_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.copilot_sessions
    ADD CONSTRAINT copilot_sessions_pkey PRIMARY KEY (id);


--
-- Name: credentials credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.credentials
    ADD CONSTRAINT credentials_pkey PRIMARY KEY (id);


--
-- Name: edges edges_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edges
    ADD CONSTRAINT edges_pkey PRIMARY KEY (id);


--
-- Name: edges edges_source_step_id_target_step_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edges
    ADD CONSTRAINT edges_source_step_id_target_step_id_key UNIQUE (source_step_id, target_step_id);


--
-- Name: runs runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.runs
    ADD CONSTRAINT runs_pkey PRIMARY KEY (id);


--
-- Name: schedules schedules_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schedules
    ADD CONSTRAINT schedules_pkey PRIMARY KEY (id);


--
-- Name: secrets secrets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_pkey PRIMARY KEY (id);


--
-- Name: secrets secrets_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: step_runs step_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.step_runs
    ADD CONSTRAINT step_runs_pkey PRIMARY KEY (id);


--
-- Name: steps steps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.steps
    ADD CONSTRAINT steps_pkey PRIMARY KEY (id);


--
-- Name: system_credentials system_credentials_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_credentials
    ADD CONSTRAINT system_credentials_name_key UNIQUE (name);


--
-- Name: system_credentials system_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_credentials
    ADD CONSTRAINT system_credentials_pkey PRIMARY KEY (id);


--
-- Name: tenants tenants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_pkey PRIMARY KEY (id);


--
-- Name: tenants tenants_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_slug_key UNIQUE (slug);


--
-- Name: block_definitions unique_block_slug; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_definitions
    ADD CONSTRAINT unique_block_slug UNIQUE NULLS NOT DISTINCT (tenant_id, slug);


--
-- Name: credentials unique_credential_name; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.credentials
    ADD CONSTRAINT unique_credential_name UNIQUE (tenant_id, name);


--
-- Name: usage_budgets usage_budgets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_budgets
    ADD CONSTRAINT usage_budgets_pkey PRIMARY KEY (id);


--
-- Name: usage_daily_aggregates usage_daily_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_daily_aggregates
    ADD CONSTRAINT usage_daily_aggregates_pkey PRIMARY KEY (id);


--
-- Name: usage_records usage_records_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_records
    ADD CONSTRAINT usage_records_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_tenant_id_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_tenant_id_email_key UNIQUE (tenant_id, email);


--
-- Name: webhooks webhooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_pkey PRIMARY KEY (id);


--
-- Name: workflow_versions workflow_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_versions
    ADD CONSTRAINT workflow_versions_pkey PRIMARY KEY (id);


--
-- Name: workflow_versions workflow_versions_workflow_id_version_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_versions
    ADD CONSTRAINT workflow_versions_workflow_id_version_key UNIQUE (workflow_id, version);


--
-- Name: workflows workflows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_pkey PRIMARY KEY (id);


--
-- Name: idx_audit_logs_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_logs_created ON public.audit_logs USING btree (created_at);


--
-- Name: idx_audit_logs_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_logs_tenant ON public.audit_logs USING btree (tenant_id);


--
-- Name: idx_block_definitions_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_block_definitions_category ON public.block_definitions USING btree (category);


--
-- Name: idx_block_definitions_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_block_definitions_enabled ON public.block_definitions USING btree (enabled) WHERE (enabled = true);


--
-- Name: idx_block_definitions_slug; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_block_definitions_slug ON public.block_definitions USING btree (slug);


--
-- Name: idx_block_definitions_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_block_definitions_tenant ON public.block_definitions USING btree (tenant_id);


--
-- Name: idx_block_group_runs_block_group; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_block_group_runs_block_group ON public.block_group_runs USING btree (block_group_id);


--
-- Name: idx_block_group_runs_run; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_block_group_runs_run ON public.block_group_runs USING btree (run_id);


--
-- Name: idx_block_groups_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_block_groups_parent ON public.block_groups USING btree (parent_group_id);


--
-- Name: idx_block_groups_workflow; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_block_groups_workflow ON public.block_groups USING btree (workflow_id);


--
-- Name: idx_block_versions_block_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_block_versions_block_id ON public.block_versions USING btree (block_id);


--
-- Name: idx_block_versions_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_block_versions_created_at ON public.block_versions USING btree (created_at);


--
-- Name: idx_copilot_messages_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_copilot_messages_created ON public.copilot_messages USING btree (session_id, created_at);


--
-- Name: idx_copilot_messages_session; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_copilot_messages_session ON public.copilot_messages USING btree (session_id);


--
-- Name: idx_copilot_sessions_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_copilot_sessions_active ON public.copilot_sessions USING btree (tenant_id, user_id, workflow_id, is_active) WHERE (is_active = true);


--
-- Name: idx_copilot_sessions_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_copilot_sessions_tenant ON public.copilot_sessions USING btree (tenant_id);


--
-- Name: idx_copilot_sessions_user_workflow; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_copilot_sessions_user_workflow ON public.copilot_sessions USING btree (tenant_id, user_id, workflow_id);


--
-- Name: idx_credentials_expires; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_credentials_expires ON public.credentials USING btree (expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: idx_credentials_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_credentials_status ON public.credentials USING btree (status);


--
-- Name: idx_credentials_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_credentials_tenant ON public.credentials USING btree (tenant_id);


--
-- Name: idx_credentials_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_credentials_type ON public.credentials USING btree (credential_type);


--
-- Name: idx_edges_source_port; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_edges_source_port ON public.edges USING btree (source_step_id, source_port);


--
-- Name: idx_edges_target_port; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_edges_target_port ON public.edges USING btree (target_port) WHERE ((target_port)::text <> ''::text);


--
-- Name: idx_runs_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_runs_status ON public.runs USING btree (status);


--
-- Name: idx_runs_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_runs_tenant ON public.runs USING btree (tenant_id);


--
-- Name: idx_runs_trigger_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_runs_trigger_source ON public.runs USING btree (trigger_source) WHERE (trigger_source IS NOT NULL);


--
-- Name: idx_runs_workflow; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_runs_workflow ON public.runs USING btree (workflow_id);


--
-- Name: idx_schedules_next_run; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_schedules_next_run ON public.schedules USING btree (next_run_at) WHERE ((status)::text = 'active'::text);


--
-- Name: idx_schedules_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_schedules_tenant ON public.schedules USING btree (tenant_id);


--
-- Name: idx_step_runs_run; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_step_runs_run ON public.step_runs USING btree (run_id);


--
-- Name: idx_steps_block_group; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_steps_block_group ON public.steps USING btree (block_group_id);


--
-- Name: idx_system_credentials_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_credentials_status ON public.system_credentials USING btree (status);


--
-- Name: idx_system_credentials_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_credentials_type ON public.system_credentials USING btree (credential_type);


--
-- Name: idx_tenants_owner_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenants_owner_email ON public.tenants USING btree (owner_email);


--
-- Name: idx_tenants_plan; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenants_plan ON public.tenants USING btree (plan);


--
-- Name: idx_tenants_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenants_status ON public.tenants USING btree (status);


--
-- Name: idx_usage_budgets_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_budgets_tenant ON public.usage_budgets USING btree (tenant_id);


--
-- Name: idx_usage_budgets_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_usage_budgets_unique ON public.usage_budgets USING btree (tenant_id, COALESCE(workflow_id, '00000000-0000-0000-0000-000000000000'::uuid), budget_type);


--
-- Name: idx_usage_budgets_workflow; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_budgets_workflow ON public.usage_budgets USING btree (workflow_id);


--
-- Name: idx_usage_daily_tenant_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_daily_tenant_date ON public.usage_daily_aggregates USING btree (tenant_id, date);


--
-- Name: idx_usage_daily_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_usage_daily_unique ON public.usage_daily_aggregates USING btree (tenant_id, COALESCE(workflow_id, '00000000-0000-0000-0000-000000000000'::uuid), date, provider, model);


--
-- Name: idx_usage_daily_workflow; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_daily_workflow ON public.usage_daily_aggregates USING btree (workflow_id);


--
-- Name: idx_usage_records_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_records_created_at ON public.usage_records USING btree (created_at);


--
-- Name: idx_usage_records_provider_model; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_records_provider_model ON public.usage_records USING btree (provider, model);


--
-- Name: idx_usage_records_run; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_records_run ON public.usage_records USING btree (run_id);


--
-- Name: idx_usage_records_tenant_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_records_tenant_date ON public.usage_records USING btree (tenant_id, created_at);


--
-- Name: idx_usage_records_workflow; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_records_workflow ON public.usage_records USING btree (workflow_id);


--
-- Name: idx_workflows_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflows_status ON public.workflows USING btree (status);


--
-- Name: idx_workflows_system_slug; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_workflows_system_slug ON public.workflows USING btree (system_slug) WHERE (system_slug IS NOT NULL);


--
-- Name: idx_workflows_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflows_tenant ON public.workflows USING btree (tenant_id);


--
-- Trigger Functions
--

CREATE OR REPLACE FUNCTION public.update_block_definitions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION public.update_credentials_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION public.update_system_credentials_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

--
-- Name: block_definitions trigger_block_definitions_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_block_definitions_updated_at BEFORE UPDATE ON public.block_definitions FOR EACH ROW EXECUTE FUNCTION public.update_block_definitions_updated_at();


--
-- Name: credentials trigger_credentials_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_credentials_updated_at BEFORE UPDATE ON public.credentials FOR EACH ROW EXECUTE FUNCTION public.update_credentials_updated_at();


--
-- Name: system_credentials trigger_system_credentials_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_system_credentials_updated_at BEFORE UPDATE ON public.system_credentials FOR EACH ROW EXECUTE FUNCTION public.update_system_credentials_updated_at();


--
-- Name: adapters adapters_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.adapters
    ADD CONSTRAINT adapters_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);


--
-- Name: audit_logs audit_logs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);


--
-- Name: block_definitions block_definitions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_definitions
    ADD CONSTRAINT block_definitions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: block_group_runs block_group_runs_block_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_group_runs
    ADD CONSTRAINT block_group_runs_block_group_id_fkey FOREIGN KEY (block_group_id) REFERENCES public.block_groups(id) ON DELETE CASCADE;


--
-- Name: block_group_runs block_group_runs_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_group_runs
    ADD CONSTRAINT block_group_runs_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.runs(id) ON DELETE CASCADE;


--
-- Name: block_groups block_groups_parent_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_groups
    ADD CONSTRAINT block_groups_parent_group_id_fkey FOREIGN KEY (parent_group_id) REFERENCES public.block_groups(id) ON DELETE CASCADE;


--
-- Name: block_groups block_groups_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_groups
    ADD CONSTRAINT block_groups_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: block_versions block_versions_block_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.block_versions
    ADD CONSTRAINT block_versions_block_id_fkey FOREIGN KEY (block_id) REFERENCES public.block_definitions(id) ON DELETE CASCADE;


--
-- Name: copilot_messages copilot_messages_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.copilot_messages
    ADD CONSTRAINT copilot_messages_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.copilot_sessions(id) ON DELETE CASCADE;


--
-- Name: copilot_sessions copilot_sessions_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.copilot_sessions
    ADD CONSTRAINT copilot_sessions_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: credentials credentials_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.credentials
    ADD CONSTRAINT credentials_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: edges edges_source_step_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edges
    ADD CONSTRAINT edges_source_step_id_fkey FOREIGN KEY (source_step_id) REFERENCES public.steps(id) ON DELETE CASCADE;


--
-- Name: edges edges_target_step_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edges
    ADD CONSTRAINT edges_target_step_id_fkey FOREIGN KEY (target_step_id) REFERENCES public.steps(id) ON DELETE CASCADE;


--
-- Name: edges edges_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edges
    ADD CONSTRAINT edges_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: runs runs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.runs
    ADD CONSTRAINT runs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);


--
-- Name: runs runs_triggered_by_user_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.runs
    ADD CONSTRAINT runs_triggered_by_user_fkey FOREIGN KEY (triggered_by_user) REFERENCES public.users(id);


--
-- Name: runs runs_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.runs
    ADD CONSTRAINT runs_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: schedules schedules_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schedules
    ADD CONSTRAINT schedules_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: schedules schedules_last_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schedules
    ADD CONSTRAINT schedules_last_run_id_fkey FOREIGN KEY (last_run_id) REFERENCES public.runs(id);


--
-- Name: schedules schedules_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schedules
    ADD CONSTRAINT schedules_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);


--
-- Name: schedules schedules_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schedules
    ADD CONSTRAINT schedules_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: secrets secrets_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: secrets secrets_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);


--
-- Name: step_runs step_runs_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.step_runs
    ADD CONSTRAINT step_runs_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.runs(id) ON DELETE CASCADE;


--
-- Name: steps steps_block_definition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.steps
    ADD CONSTRAINT steps_block_definition_id_fkey FOREIGN KEY (block_definition_id) REFERENCES public.block_definitions(id) ON DELETE SET NULL;


--
-- Name: steps steps_block_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.steps
    ADD CONSTRAINT steps_block_group_id_fkey FOREIGN KEY (block_group_id) REFERENCES public.block_groups(id) ON DELETE SET NULL;


--
-- Name: steps steps_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.steps
    ADD CONSTRAINT steps_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: usage_budgets usage_budgets_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_budgets
    ADD CONSTRAINT usage_budgets_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);


--
-- Name: usage_budgets usage_budgets_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_budgets
    ADD CONSTRAINT usage_budgets_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: usage_daily_aggregates usage_daily_aggregates_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_daily_aggregates
    ADD CONSTRAINT usage_daily_aggregates_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);


--
-- Name: usage_daily_aggregates usage_daily_aggregates_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_daily_aggregates
    ADD CONSTRAINT usage_daily_aggregates_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: usage_records usage_records_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_records
    ADD CONSTRAINT usage_records_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.runs(id);


--
-- Name: usage_records usage_records_step_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_records
    ADD CONSTRAINT usage_records_step_run_id_fkey FOREIGN KEY (step_run_id) REFERENCES public.step_runs(id);


--
-- Name: usage_records usage_records_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_records
    ADD CONSTRAINT usage_records_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);


--
-- Name: usage_records usage_records_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_records
    ADD CONSTRAINT usage_records_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: users users_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);


--
-- Name: webhooks webhooks_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: webhooks webhooks_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);


--
-- Name: webhooks webhooks_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_versions workflow_versions_saved_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_versions
    ADD CONSTRAINT workflow_versions_saved_by_fkey FOREIGN KEY (saved_by) REFERENCES public.users(id);


--
-- Name: workflow_versions workflow_versions_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_versions
    ADD CONSTRAINT workflow_versions_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflows workflows_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: workflows workflows_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);


--
-- PostgreSQL database dump complete
--

-- ============================================================================
-- RAG (Retrieval-Augmented Generation) Tables
-- ============================================================================

--
-- Name: vector; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS vector;


--
-- Name: vector_collections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vector_collections (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    embedding_provider character varying(50) DEFAULT 'openai'::character varying NOT NULL,
    embedding_model character varying(100) DEFAULT 'text-embedding-3-small'::character varying NOT NULL,
    dimension integer DEFAULT 1536 NOT NULL,
    document_count integer DEFAULT 0,
    metadata jsonb DEFAULT '{}'::jsonb,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: TABLE vector_collections; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.vector_collections IS 'RAG vector collections with tenant isolation';


--
-- Name: COLUMN vector_collections.dimension; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.vector_collections.dimension IS 'Vector dimension (1536 for text-embedding-3-small, 3072 for text-embedding-3-large)';


--
-- Name: vector_documents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vector_documents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    collection_id uuid NOT NULL,
    content text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    embedding public.vector(1536),
    source_url text,
    source_type character varying(50),
    chunk_index integer,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: TABLE vector_documents; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.vector_documents IS 'RAG vector documents with embeddings, tenant-isolated';


--
-- Name: COLUMN vector_documents.content; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.vector_documents.content IS 'Document content (LangChain Document.page_content equivalent)';


--
-- Name: COLUMN vector_documents.metadata; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.vector_documents.metadata IS 'Document metadata (LangChain Document.metadata equivalent)';


--
-- Name: COLUMN vector_documents.embedding; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.vector_documents.embedding IS 'Vector embedding from embedding model';


--
-- Name: COLUMN vector_documents.source_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.vector_documents.source_type IS 'Source type: url, file, text, api';


--
-- Name: vector_collections vector_collections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vector_collections
    ADD CONSTRAINT vector_collections_pkey PRIMARY KEY (id);


--
-- Name: vector_collections unique_collection_per_tenant; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vector_collections
    ADD CONSTRAINT unique_collection_per_tenant UNIQUE (tenant_id, name);


--
-- Name: vector_documents vector_documents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vector_documents
    ADD CONSTRAINT vector_documents_pkey PRIMARY KEY (id);


--
-- Name: idx_vector_collections_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vector_collections_tenant ON public.vector_collections USING btree (tenant_id);


--
-- Name: idx_vector_documents_tenant_collection; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vector_documents_tenant_collection ON public.vector_documents USING btree (tenant_id, collection_id);


--
-- Name: idx_vector_documents_embedding; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vector_documents_embedding ON public.vector_documents USING ivfflat (embedding public.vector_cosine_ops) WITH (lists = 100);


--
-- Name: idx_vector_documents_metadata; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vector_documents_metadata ON public.vector_documents USING gin (metadata);


--
-- Name: vector_collections trigger_vector_collections_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE OR REPLACE FUNCTION public.update_vector_collections_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_vector_collections_updated_at BEFORE UPDATE ON public.vector_collections FOR EACH ROW EXECUTE FUNCTION public.update_vector_collections_updated_at();


--
-- Name: vector_collections vector_collections_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vector_collections
    ADD CONSTRAINT vector_collections_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: vector_documents vector_documents_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vector_documents
    ADD CONSTRAINT vector_documents_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: vector_documents vector_documents_collection_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vector_documents
    ADD CONSTRAINT vector_documents_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES public.vector_collections(id) ON DELETE CASCADE;


\unrestrict qS68oUh7R0Ylf9hxaFCN1NsqAxjwYu48PFRRQDDRLom2r3jbe88asrCBhclQEWj

