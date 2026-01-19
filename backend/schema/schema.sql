-- AI Orchestration Database Schema
--
-- Usage:
--   make db-apply   - Apply this schema to database
--   make db-reset   - Drop and recreate all tables
--   make db-seed    - Load initial data
--
-- This file is the single source of truth for the database schema.
--
-- MAJOR CHANGE: Workflow → Project with Multi-Start model
-- - workflows table → projects table
-- - workflow_versions → project_versions
-- - webhooks table → removed (integrated into steps.trigger_config)
-- - steps.workflow_id → steps.project_id
-- - steps now has trigger_type/trigger_config for Start blocks

--
-- PostgreSQL database dump
--

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

-- ============================================================================
-- Extension: uuid-ossp
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- Core Tables
-- ============================================================================

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
    suspended_reason text,
    variables jsonb DEFAULT '{}'::jsonb
);

COMMENT ON COLUMN public.tenants.status IS 'Tenant status: active, suspended, pending, inactive';
COMMENT ON COLUMN public.tenants.plan IS 'Subscription plan: free, starter, professional, enterprise';
COMMENT ON COLUMN public.tenants.owner_email IS 'Primary contact email for the tenant';
COMMENT ON COLUMN public.tenants.owner_name IS 'Primary contact name for the tenant';
COMMENT ON COLUMN public.tenants.billing_email IS 'Email for billing notifications';
COMMENT ON COLUMN public.tenants.metadata IS 'Additional tenant metadata (industry, company_size, website, country, notes)';
COMMENT ON COLUMN public.tenants.feature_flags IS 'Feature flags: copilot_enabled, advanced_analytics, custom_blocks, api_access, sso_enabled, audit_logs, max_concurrent_runs';
COMMENT ON COLUMN public.tenants.limits IS 'Resource limits: max_projects, max_runs_per_day, max_users, max_credentials, max_storage_mb, retention_days';
COMMENT ON COLUMN public.tenants.suspended_at IS 'Timestamp when tenant was suspended';
COMMENT ON COLUMN public.tenants.suspended_reason IS 'Reason for tenant suspension';
COMMENT ON COLUMN public.tenants.variables IS 'Organization variables accessible by {{$org.xxx}} in templates';

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
    updated_at timestamp with time zone DEFAULT now(),
    variables jsonb DEFAULT '{}'::jsonb
);

COMMENT ON COLUMN public.users.variables IS 'Personal variables accessible by {{$personal.xxx}} in templates';

-- ============================================================================
-- Projects (formerly Workflows) - Multi-Start Model
-- ============================================================================

--
-- Name: projects; Type: TABLE; Schema: public; Owner: -
-- NOTE: This replaces the workflows table. Projects contain multiple Start blocks.
--

CREATE TABLE public.projects (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    status character varying(50) DEFAULT 'draft'::character varying NOT NULL,
    version integer DEFAULT 0 NOT NULL,
    variables jsonb DEFAULT '{}'::jsonb,
    draft jsonb,
    is_system boolean DEFAULT false NOT NULL,
    system_slug character varying(100),
    created_by uuid,
    published_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    CONSTRAINT projects_status_check CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'published'::character varying])::text[])))
);

COMMENT ON TABLE public.projects IS 'Projects contain DAGs with multiple Start blocks (entry points). Replaces workflows table.';
COMMENT ON COLUMN public.projects.variables IS 'Shared variables accessible by all steps in the project';
COMMENT ON COLUMN public.projects.is_system IS 'True for system projects (e.g., Copilot). These are accessible across all tenants.';
COMMENT ON COLUMN public.projects.system_slug IS 'Unique slug for system projects (e.g., copilot-generate). Used for internal lookups.';

--
-- Name: project_versions; Type: TABLE; Schema: public; Owner: -
-- NOTE: This replaces workflow_versions table.
--

CREATE TABLE public.project_versions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    version integer NOT NULL,
    definition jsonb NOT NULL,
    saved_by uuid,
    saved_at timestamp with time zone DEFAULT now()
);

COMMENT ON TABLE public.project_versions IS 'Version history for projects (immutable snapshots)';

-- ============================================================================
-- Steps, Edges, Block Groups
-- ============================================================================

--
-- Name: block_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.block_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    type character varying(50) NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    parent_group_id uuid,
    position_x integer DEFAULT 0,
    position_y integer DEFAULT 0,
    width integer DEFAULT 400,
    height integer DEFAULT 300,
    pre_process text,
    post_process text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT valid_block_group_type CHECK (((type)::text = ANY ((ARRAY['parallel'::character varying, 'try_catch'::character varying, 'foreach'::character varying, 'while'::character varying, 'agent'::character varying])::text[])))
);

COMMENT ON TABLE public.block_groups IS 'Control flow constructs that group multiple steps';
COMMENT ON COLUMN public.block_groups.type IS 'Type of control flow: parallel, try_catch, foreach, while';
COMMENT ON COLUMN public.block_groups.config IS 'Type-specific configuration (JSON)';
COMMENT ON COLUMN public.block_groups.parent_group_id IS 'Reference to parent group for nested structures';

--
-- Name: steps; Type: TABLE; Schema: public; Owner: -
-- NOTE: workflow_id → project_id, added trigger_type/trigger_config for Start blocks
--

CREATE TABLE public.steps (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    type character varying(50) NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    trigger_type character varying(50),
    trigger_config jsonb DEFAULT '{}'::jsonb,
    position_x integer DEFAULT 0,
    position_y integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    block_group_id uuid,
    group_role character varying(50),
    credential_bindings jsonb DEFAULT '{}'::jsonb,
    block_definition_id uuid,
    CONSTRAINT steps_trigger_type_check CHECK ((trigger_type IS NULL OR (trigger_type)::text = ANY ((ARRAY['manual'::character varying, 'webhook'::character varying, 'schedule'::character varying, 'slack'::character varying, 'discord'::character varying, 'email'::character varying, 'internal'::character varying, 'api'::character varying])::text[])))
);

COMMENT ON COLUMN public.steps.project_id IS 'Reference to parent project (formerly workflow_id)';
COMMENT ON COLUMN public.steps.block_group_id IS 'Reference to containing block group (NULL if not in a group)';
COMMENT ON COLUMN public.steps.group_role IS 'Role within block group: body (steps inside the group body)';
COMMENT ON COLUMN public.steps.credential_bindings IS 'Mapping of credential names to tenant credential IDs';
COMMENT ON COLUMN public.steps.block_definition_id IS 'Reference to block_definitions registry';
COMMENT ON COLUMN public.steps.trigger_type IS 'For Start blocks: manual, webhook, schedule, slack, discord, email, internal, api';
COMMENT ON COLUMN public.steps.trigger_config IS 'For Start blocks: trigger-specific configuration (secret, cron, input_mapping, etc.)';

--
-- Name: edges; Type: TABLE; Schema: public; Owner: -
-- NOTE: workflow_id → project_id
--

CREATE TABLE public.edges (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    source_step_id uuid,
    target_step_id uuid,
    source_block_group_id uuid,
    target_block_group_id uuid,
    condition text,
    created_at timestamp with time zone DEFAULT now(),
    source_port character varying(50) DEFAULT ''::character varying,
    target_port character varying(50) DEFAULT ''::character varying,
    CONSTRAINT edges_source_check CHECK ((source_step_id IS NOT NULL OR source_block_group_id IS NOT NULL)),
    CONSTRAINT edges_target_check CHECK ((target_step_id IS NOT NULL OR target_block_group_id IS NOT NULL))
);

COMMENT ON COLUMN public.edges.project_id IS 'Reference to parent project (formerly workflow_id)';

-- ============================================================================
-- Block Definitions (Block Registry)
-- ============================================================================

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
    subcategory character varying(50),
    icon character varying(50),
    config_schema jsonb DEFAULT '{}'::jsonb NOT NULL,
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
    parent_block_id uuid,
    config_defaults jsonb DEFAULT '{}'::jsonb,
    pre_process text,
    post_process text,
    internal_steps jsonb DEFAULT '[]'::jsonb,
    group_kind character varying(50),
    is_container boolean DEFAULT false NOT NULL,
    CONSTRAINT valid_block_category CHECK (((category)::text = ANY ((ARRAY['ai'::character varying, 'flow'::character varying, 'apps'::character varying, 'custom'::character varying])::text[]))),
    CONSTRAINT valid_block_subcategory CHECK (subcategory IS NULL OR (subcategory)::text = ANY ((ARRAY['chat'::character varying, 'rag'::character varying, 'routing'::character varying, 'branching'::character varying, 'data'::character varying, 'control'::character varying, 'utility'::character varying, 'slack'::character varying, 'discord'::character varying, 'notion'::character varying, 'github'::character varying, 'google'::character varying, 'linear'::character varying, 'email'::character varying, 'web'::character varying, 'agent'::character varying])::text[])),
    CONSTRAINT valid_group_kind CHECK (group_kind IS NULL OR (group_kind)::text = ANY ((ARRAY['parallel'::character varying, 'try_catch'::character varying, 'foreach'::character varying, 'while'::character varying, 'agent'::character varying])::text[])),
    CONSTRAINT no_self_reference CHECK (parent_block_id IS NULL OR parent_block_id != id)
);

COMMENT ON COLUMN public.block_definitions.required_credentials IS 'JSON array declaring required credentials: [{name, type, scope, description, required}]';
COMMENT ON COLUMN public.block_definitions.is_public IS 'Whether tenant block is visible to other tenants';
COMMENT ON COLUMN public.block_definitions.code IS 'JavaScript code executed in sandbox. All blocks are code-based.';
COMMENT ON COLUMN public.block_definitions.ui_config IS 'UI metadata: icon, color, configSchema for project editor';
COMMENT ON COLUMN public.block_definitions.is_system IS 'System blocks can only be edited by admins';
COMMENT ON COLUMN public.block_definitions.version IS 'Version number, incremented on each update';
COMMENT ON COLUMN public.block_definitions.parent_block_id IS 'Parent block for inheritance (only blocks with code can be inherited)';
COMMENT ON COLUMN public.block_definitions.config_defaults IS 'Default values for parent config_schema when inheriting';
COMMENT ON COLUMN public.block_definitions.pre_process IS 'JavaScript code executed before main code (input transformation)';
COMMENT ON COLUMN public.block_definitions.post_process IS 'JavaScript code executed after main code (output transformation)';
COMMENT ON COLUMN public.block_definitions.internal_steps IS 'Array of internal steps to execute sequentially: [{type, config, output_key}]';

--
-- Name: block_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.block_versions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    block_id uuid NOT NULL,
    version integer NOT NULL,
    code text NOT NULL,
    config_schema jsonb NOT NULL,
    output_schema jsonb,
    ui_config jsonb NOT NULL,
    change_summary text,
    changed_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE public.block_versions IS 'Version history for block definitions, enables rollback';

-- ============================================================================
-- Execution (Runs, Step Runs, Block Group Runs)
-- ============================================================================

--
-- Name: runs; Type: TABLE; Schema: public; Owner: -
-- NOTE: workflow_id → project_id, added start_step_id
--

CREATE TABLE public.runs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    project_version integer NOT NULL,
    start_step_id uuid,
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    input jsonb,
    output jsonb,
    error text,
    triggered_by character varying(50) DEFAULT 'manual'::character varying NOT NULL,
    run_number integer DEFAULT 0 NOT NULL,
    triggered_by_user uuid,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    trigger_source character varying(100),
    trigger_metadata jsonb DEFAULT '{}'::jsonb,
    deleted_at timestamp with time zone
);

COMMENT ON COLUMN public.runs.project_id IS 'Reference to parent project (formerly workflow_id)';
COMMENT ON COLUMN public.runs.project_version IS 'Project version that was executed (formerly workflow_version)';
COMMENT ON COLUMN public.runs.start_step_id IS 'Which Start block triggered this run';
COMMENT ON COLUMN public.runs.trigger_source IS 'Internal trigger source identifier: copilot, audit-system, etc.';
COMMENT ON COLUMN public.runs.trigger_metadata IS 'Additional metadata about the trigger: feature, user_id, session_id, etc.';
COMMENT ON COLUMN public.runs.run_number IS 'Sequential run number per project + triggered_by combination';

--
-- Name: run_number_sequences; Type: TABLE; Schema: public; Owner: -
-- NOTE: workflow_id → project_id
--

CREATE TABLE public.run_number_sequences (
    project_id uuid NOT NULL,
    triggered_by character varying(50) NOT NULL,
    next_number integer DEFAULT 1 NOT NULL
);

COMMENT ON TABLE public.run_number_sequences IS 'Tracks next run_number for each project + triggered_by combination';

--
-- Name: step_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.step_runs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    run_id uuid NOT NULL,
    step_id uuid NOT NULL,
    step_name character varying(255) NOT NULL,
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    attempt integer DEFAULT 1 NOT NULL,
    sequence_number integer DEFAULT 0 NOT NULL,
    input jsonb,
    output jsonb,
    error text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    duration_ms integer,
    created_at timestamp with time zone DEFAULT now()
);

COMMENT ON COLUMN public.step_runs.sequence_number IS 'Execution order within the same run and attempt (1-indexed)';

-- ============================================================================
-- Scheduling
-- ============================================================================

--
-- Name: schedules; Type: TABLE; Schema: public; Owner: -
-- NOTE: workflow_id → project_id, added start_step_id
--

CREATE TABLE public.schedules (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    start_step_id uuid NOT NULL,
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

COMMENT ON COLUMN public.schedules.project_id IS 'Reference to parent project (formerly workflow_id)';
COMMENT ON COLUMN public.schedules.start_step_id IS 'Which Start block to execute when schedule triggers';

-- ============================================================================
-- Credentials & Secrets
-- ============================================================================

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

COMMENT ON TABLE public.credentials IS 'Stores encrypted API credentials for external service authentication';
COMMENT ON COLUMN public.credentials.encrypted_data IS 'AES-256-GCM encrypted credential data (secrets)';
COMMENT ON COLUMN public.credentials.encrypted_dek IS 'Encrypted Data Encryption Key (envelope encryption)';
COMMENT ON COLUMN public.credentials.data_nonce IS '12-byte nonce/IV for data encryption';
COMMENT ON COLUMN public.credentials.dek_nonce IS '12-byte nonce/IV for DEK encryption';
COMMENT ON COLUMN public.credentials.metadata IS 'Non-sensitive metadata (e.g., service name, account info)';

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

COMMENT ON TABLE public.system_credentials IS 'Operator-managed credentials for system blocks (not accessible by tenants)';

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

-- ============================================================================
-- Usage Tracking & Billing
-- ============================================================================

--
-- Name: usage_records; Type: TABLE; Schema: public; Owner: -
-- NOTE: workflow_id → project_id
--

CREATE TABLE public.usage_records (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid,
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

COMMENT ON TABLE public.usage_records IS 'Individual LLM API call records with token usage and cost';
COMMENT ON COLUMN public.usage_records.project_id IS 'Reference to parent project (formerly workflow_id)';
COMMENT ON COLUMN public.usage_records.provider IS 'LLM provider: openai, anthropic, google, etc.';
COMMENT ON COLUMN public.usage_records.model IS 'Model identifier: gpt-4o, claude-3-opus, etc.';
COMMENT ON COLUMN public.usage_records.operation IS 'Operation type: chat, completion, embedding, etc.';
COMMENT ON COLUMN public.usage_records.total_cost_usd IS 'Total cost in USD with 8 decimal precision';

--
-- Name: usage_daily_aggregates; Type: TABLE; Schema: public; Owner: -
-- NOTE: workflow_id → project_id
--

CREATE TABLE public.usage_daily_aggregates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid,
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

COMMENT ON TABLE public.usage_daily_aggregates IS 'Pre-aggregated daily usage data for dashboard performance';

--
-- Name: usage_budgets; Type: TABLE; Schema: public; Owner: -
-- NOTE: workflow_id → project_id
--

CREATE TABLE public.usage_budgets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid,
    budget_type character varying(50) NOT NULL,
    budget_amount_usd numeric(12,2) NOT NULL,
    alert_threshold numeric(3,2) DEFAULT 0.80 NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE public.usage_budgets IS 'Budget settings for cost control and alerts';
COMMENT ON COLUMN public.usage_budgets.alert_threshold IS 'Percentage (0.00-1.00) at which to trigger alert';

-- ============================================================================
-- Audit
-- ============================================================================

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

-- ============================================================================
-- Copilot (AI Workflow Assistant)
-- NOTE: Unified copilot system - integrated into workflow editor
-- ============================================================================

--
-- Name: copilot_sessions; Type: TABLE; Schema: public; Owner: -
-- Copilot sessions for AI-assisted workflow creation/enhancement
--

CREATE TABLE public.copilot_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    user_id character varying(255) NOT NULL,

    -- Context: which workflow this session is scoped to
    context_project_id uuid,

    -- Mode: create (new workflow), enhance (improve existing), explain (understand)
    mode character varying(50) DEFAULT 'create'::character varying NOT NULL,

    -- Title: derived from first user message
    title character varying(200),

    -- Status
    status character varying(50) DEFAULT 'hearing'::character varying NOT NULL,
    hearing_phase character varying(50) DEFAULT 'analysis'::character varying NOT NULL,
    hearing_progress integer DEFAULT 0 NOT NULL,

    -- Generated artifacts
    spec jsonb,
    project_id uuid,

    -- Timestamps
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,

    CONSTRAINT copilot_sessions_mode_check CHECK (((mode)::text = ANY ((ARRAY['create'::character varying, 'enhance'::character varying, 'explain'::character varying])::text[]))),
    CONSTRAINT copilot_sessions_status_check CHECK (((status)::text = ANY ((ARRAY['hearing'::character varying, 'building'::character varying, 'reviewing'::character varying, 'refining'::character varying, 'completed'::character varying, 'abandoned'::character varying])::text[]))),
    CONSTRAINT copilot_sessions_phase_check CHECK (((hearing_phase)::text = ANY ((ARRAY['analysis'::character varying, 'proposal'::character varying, 'completed'::character varying])::text[]))),
    CONSTRAINT copilot_sessions_progress_check CHECK ((hearing_progress >= 0 AND hearing_progress <= 100))
);

COMMENT ON TABLE public.copilot_sessions IS 'AI Copilot sessions for interactive workflow creation/enhancement';
COMMENT ON COLUMN public.copilot_sessions.context_project_id IS 'The workflow this session is scoped to (NULL for global create)';
COMMENT ON COLUMN public.copilot_sessions.mode IS 'Copilot mode: create (new workflow), enhance (improve existing), explain (understand)';
COMMENT ON COLUMN public.copilot_sessions.status IS 'Session status: hearing, building, reviewing, refining, completed, abandoned';
COMMENT ON COLUMN public.copilot_sessions.hearing_phase IS 'Current hearing phase: analysis, proposal, completed';
COMMENT ON COLUMN public.copilot_sessions.spec IS 'WorkflowSpec DSL as JSON';
COMMENT ON COLUMN public.copilot_sessions.project_id IS 'Generated/modified project ID after construction';

--
-- Name: copilot_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.copilot_messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    session_id uuid NOT NULL,
    role character varying(20) NOT NULL,
    content text NOT NULL,

    -- Metadata
    phase character varying(50),
    extracted_data jsonb,
    suggested_questions jsonb,

    -- Timestamps
    created_at timestamp with time zone DEFAULT now() NOT NULL,

    CONSTRAINT copilot_messages_role_check CHECK (((role)::text = ANY ((ARRAY['user'::character varying, 'assistant'::character varying, 'system'::character varying])::text[])))
);

COMMENT ON TABLE public.copilot_messages IS 'Messages in AI Copilot sessions';
COMMENT ON COLUMN public.copilot_messages.phase IS 'Hearing phase when this message was created';
COMMENT ON COLUMN public.copilot_messages.extracted_data IS 'Data extracted from user message';
COMMENT ON COLUMN public.copilot_messages.suggested_questions IS 'Suggested follow-up questions';

-- ============================================================================
-- Primary Keys
-- ============================================================================

ALTER TABLE ONLY public.tenants ADD CONSTRAINT tenants_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tenants ADD CONSTRAINT tenants_slug_key UNIQUE (slug);

ALTER TABLE ONLY public.users ADD CONSTRAINT users_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.users ADD CONSTRAINT users_tenant_id_email_key UNIQUE (tenant_id, email);

ALTER TABLE ONLY public.projects ADD CONSTRAINT projects_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.project_versions ADD CONSTRAINT project_versions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.project_versions ADD CONSTRAINT project_versions_project_id_version_key UNIQUE (project_id, version);

ALTER TABLE ONLY public.steps ADD CONSTRAINT steps_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.edges ADD CONSTRAINT edges_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.block_groups ADD CONSTRAINT block_groups_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.block_definitions ADD CONSTRAINT block_definitions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.block_definitions ADD CONSTRAINT unique_block_slug UNIQUE NULLS NOT DISTINCT (tenant_id, slug);
ALTER TABLE ONLY public.block_versions ADD CONSTRAINT block_versions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.block_versions ADD CONSTRAINT block_versions_block_id_version_key UNIQUE (block_id, version);

ALTER TABLE ONLY public.runs ADD CONSTRAINT runs_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.run_number_sequences ADD CONSTRAINT run_number_sequences_pkey PRIMARY KEY (project_id, triggered_by);
ALTER TABLE ONLY public.step_runs ADD CONSTRAINT step_runs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.schedules ADD CONSTRAINT schedules_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.credentials ADD CONSTRAINT credentials_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.credentials ADD CONSTRAINT unique_credential_name UNIQUE (tenant_id, name);
ALTER TABLE ONLY public.system_credentials ADD CONSTRAINT system_credentials_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.system_credentials ADD CONSTRAINT system_credentials_name_key UNIQUE (name);
ALTER TABLE ONLY public.secrets ADD CONSTRAINT secrets_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.secrets ADD CONSTRAINT secrets_tenant_id_name_key UNIQUE (tenant_id, name);

ALTER TABLE ONLY public.usage_records ADD CONSTRAINT usage_records_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.usage_daily_aggregates ADD CONSTRAINT usage_daily_aggregates_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.usage_budgets ADD CONSTRAINT usage_budgets_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.audit_logs ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.copilot_sessions ADD CONSTRAINT copilot_sessions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.copilot_messages ADD CONSTRAINT copilot_messages_pkey PRIMARY KEY (id);

-- ============================================================================
-- Unique Indexes
-- ============================================================================

CREATE UNIQUE INDEX idx_projects_system_slug ON public.projects USING btree (system_slug) WHERE (system_slug IS NOT NULL);

CREATE UNIQUE INDEX edges_unique_connection ON public.edges (
    project_id,
    COALESCE(source_step_id, '00000000-0000-0000-0000-000000000000'::uuid),
    COALESCE(target_step_id, '00000000-0000-0000-0000-000000000000'::uuid),
    COALESCE(source_block_group_id, '00000000-0000-0000-0000-000000000000'::uuid),
    COALESCE(target_block_group_id, '00000000-0000-0000-0000-000000000000'::uuid)
);

CREATE UNIQUE INDEX idx_usage_budgets_unique ON public.usage_budgets USING btree (tenant_id, COALESCE(project_id, '00000000-0000-0000-0000-000000000000'::uuid), budget_type);
CREATE UNIQUE INDEX idx_usage_daily_unique ON public.usage_daily_aggregates USING btree (tenant_id, COALESCE(project_id, '00000000-0000-0000-0000-000000000000'::uuid), date, provider, model);

-- ============================================================================
-- Indexes
-- ============================================================================

-- Tenants
CREATE INDEX idx_tenants_status ON public.tenants USING btree (status);
CREATE INDEX idx_tenants_plan ON public.tenants USING btree (plan);
CREATE INDEX idx_tenants_owner_email ON public.tenants USING btree (owner_email);

-- Projects
CREATE INDEX idx_projects_tenant ON public.projects USING btree (tenant_id);
CREATE INDEX idx_projects_status ON public.projects USING btree (status);
CREATE INDEX idx_projects_deleted ON public.projects USING btree (deleted_at) WHERE (deleted_at IS NULL);

-- Project Versions
CREATE INDEX idx_project_versions_project ON public.project_versions USING btree (project_id);

-- Steps
CREATE INDEX idx_steps_tenant ON public.steps USING btree (tenant_id);
CREATE INDEX idx_steps_project ON public.steps USING btree (project_id);
CREATE INDEX idx_steps_block_group ON public.steps USING btree (block_group_id);
CREATE INDEX idx_steps_trigger_type ON public.steps USING btree (trigger_type) WHERE (trigger_type IS NOT NULL);

-- Edges
CREATE INDEX idx_edges_tenant ON public.edges USING btree (tenant_id);
CREATE INDEX idx_edges_project ON public.edges USING btree (project_id);
CREATE INDEX idx_edges_source_port ON public.edges USING btree (source_step_id, source_port);
CREATE INDEX idx_edges_target_port ON public.edges USING btree (target_port) WHERE ((target_port)::text <> ''::text);

-- Block Groups
CREATE INDEX idx_block_groups_tenant ON public.block_groups USING btree (tenant_id);
CREATE INDEX idx_block_groups_project ON public.block_groups USING btree (project_id);
CREATE INDEX idx_block_groups_parent ON public.block_groups USING btree (parent_group_id);

-- Block Definitions
CREATE INDEX idx_block_definitions_tenant ON public.block_definitions USING btree (tenant_id);
CREATE INDEX idx_block_definitions_category ON public.block_definitions USING btree (category);
CREATE INDEX idx_block_definitions_subcategory ON public.block_definitions USING btree (subcategory) WHERE (subcategory IS NOT NULL);
CREATE INDEX idx_block_definitions_enabled ON public.block_definitions USING btree (enabled) WHERE (enabled = true);
CREATE INDEX idx_block_definitions_slug ON public.block_definitions USING btree (slug);
CREATE INDEX idx_block_definitions_parent ON public.block_definitions USING btree (parent_block_id) WHERE (parent_block_id IS NOT NULL);

-- Block Versions
CREATE INDEX idx_block_versions_block_id ON public.block_versions USING btree (block_id);
CREATE INDEX idx_block_versions_created_at ON public.block_versions USING btree (created_at);

-- Runs
CREATE INDEX idx_runs_tenant ON public.runs USING btree (tenant_id);
CREATE INDEX idx_runs_project ON public.runs USING btree (project_id);
CREATE INDEX idx_runs_status ON public.runs USING btree (status);
CREATE INDEX idx_runs_trigger_source ON public.runs USING btree (trigger_source) WHERE (trigger_source IS NOT NULL);
CREATE INDEX idx_runs_start_step ON public.runs USING btree (start_step_id) WHERE (start_step_id IS NOT NULL);

-- Step Runs
CREATE INDEX idx_step_runs_tenant ON public.step_runs USING btree (tenant_id);
CREATE INDEX idx_step_runs_run ON public.step_runs USING btree (run_id);

-- Schedules
CREATE INDEX idx_schedules_tenant ON public.schedules USING btree (tenant_id);
CREATE INDEX idx_schedules_project ON public.schedules USING btree (project_id);
CREATE INDEX idx_schedules_start_step ON public.schedules USING btree (start_step_id);
CREATE INDEX idx_schedules_next_run ON public.schedules USING btree (next_run_at) WHERE ((status)::text = 'active'::text);

-- Credentials
CREATE INDEX idx_credentials_tenant ON public.credentials USING btree (tenant_id);
CREATE INDEX idx_credentials_type ON public.credentials USING btree (credential_type);
CREATE INDEX idx_credentials_status ON public.credentials USING btree (status);
CREATE INDEX idx_credentials_expires ON public.credentials USING btree (expires_at) WHERE (expires_at IS NOT NULL);

-- System Credentials
CREATE INDEX idx_system_credentials_type ON public.system_credentials USING btree (credential_type);
CREATE INDEX idx_system_credentials_status ON public.system_credentials USING btree (status);

-- Usage
CREATE INDEX idx_usage_records_tenant_date ON public.usage_records USING btree (tenant_id, created_at);
CREATE INDEX idx_usage_records_project ON public.usage_records USING btree (project_id);
CREATE INDEX idx_usage_records_run ON public.usage_records USING btree (run_id);
CREATE INDEX idx_usage_records_provider_model ON public.usage_records USING btree (provider, model);
CREATE INDEX idx_usage_records_created_at ON public.usage_records USING btree (created_at);

CREATE INDEX idx_usage_daily_tenant_date ON public.usage_daily_aggregates USING btree (tenant_id, date);
CREATE INDEX idx_usage_daily_project ON public.usage_daily_aggregates USING btree (project_id);

CREATE INDEX idx_usage_budgets_tenant ON public.usage_budgets USING btree (tenant_id);
CREATE INDEX idx_usage_budgets_project ON public.usage_budgets USING btree (project_id);

-- Audit
CREATE INDEX idx_audit_logs_tenant ON public.audit_logs USING btree (tenant_id);
CREATE INDEX idx_audit_logs_created ON public.audit_logs USING btree (created_at);

-- Copilot
CREATE INDEX idx_copilot_sessions_tenant ON public.copilot_sessions USING btree (tenant_id);
CREATE INDEX idx_copilot_sessions_user ON public.copilot_sessions USING btree (tenant_id, user_id);
CREATE INDEX idx_copilot_sessions_status ON public.copilot_sessions USING btree (tenant_id, status);
CREATE INDEX idx_copilot_sessions_context_project ON public.copilot_sessions USING btree (context_project_id) WHERE (context_project_id IS NOT NULL);
CREATE INDEX idx_copilot_sessions_active ON public.copilot_sessions USING btree (tenant_id, user_id, status) WHERE (status NOT IN ('completed', 'abandoned'));
CREATE INDEX idx_copilot_messages_session ON public.copilot_messages USING btree (session_id);
CREATE INDEX idx_copilot_messages_created ON public.copilot_messages USING btree (session_id, created_at);

-- ============================================================================
-- Trigger Functions
-- ============================================================================

CREATE OR REPLACE FUNCTION public.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION public.assign_run_number()
RETURNS TRIGGER AS $$
BEGIN
    -- Get and increment the next run number for this project + triggered_by combination
    INSERT INTO public.run_number_sequences (project_id, triggered_by, next_number)
    VALUES (NEW.project_id, NEW.triggered_by, 2)
    ON CONFLICT (project_id, triggered_by)
    DO UPDATE SET next_number = public.run_number_sequences.next_number + 1
    RETURNING next_number - 1 INTO NEW.run_number;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Triggers
-- ============================================================================

CREATE TRIGGER trigger_projects_updated_at BEFORE UPDATE ON public.projects FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
CREATE TRIGGER trigger_steps_updated_at BEFORE UPDATE ON public.steps FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
CREATE TRIGGER trigger_block_groups_updated_at BEFORE UPDATE ON public.block_groups FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
CREATE TRIGGER trigger_block_definitions_updated_at BEFORE UPDATE ON public.block_definitions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
CREATE TRIGGER trigger_credentials_updated_at BEFORE UPDATE ON public.credentials FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
CREATE TRIGGER trigger_system_credentials_updated_at BEFORE UPDATE ON public.system_credentials FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
CREATE TRIGGER trigger_schedules_updated_at BEFORE UPDATE ON public.schedules FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
CREATE TRIGGER trigger_copilot_sessions_updated_at BEFORE UPDATE ON public.copilot_sessions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER trigger_assign_run_number BEFORE INSERT ON public.runs FOR EACH ROW EXECUTE FUNCTION public.assign_run_number();

-- ============================================================================
-- Foreign Keys
-- ============================================================================

-- Users
ALTER TABLE ONLY public.users ADD CONSTRAINT users_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

-- Projects
ALTER TABLE ONLY public.projects ADD CONSTRAINT projects_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.projects ADD CONSTRAINT projects_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

-- Project Versions
ALTER TABLE ONLY public.project_versions ADD CONSTRAINT project_versions_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.project_versions ADD CONSTRAINT project_versions_saved_by_fkey FOREIGN KEY (saved_by) REFERENCES public.users(id);

-- Steps
ALTER TABLE ONLY public.steps ADD CONSTRAINT steps_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.steps ADD CONSTRAINT steps_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.steps ADD CONSTRAINT steps_block_group_id_fkey FOREIGN KEY (block_group_id) REFERENCES public.block_groups(id) ON DELETE SET NULL;
ALTER TABLE ONLY public.steps ADD CONSTRAINT steps_block_definition_id_fkey FOREIGN KEY (block_definition_id) REFERENCES public.block_definitions(id) ON DELETE SET NULL;

-- Edges
ALTER TABLE ONLY public.edges ADD CONSTRAINT edges_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.edges ADD CONSTRAINT edges_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.edges ADD CONSTRAINT edges_source_step_id_fkey FOREIGN KEY (source_step_id) REFERENCES public.steps(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.edges ADD CONSTRAINT edges_target_step_id_fkey FOREIGN KEY (target_step_id) REFERENCES public.steps(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.edges ADD CONSTRAINT edges_source_block_group_id_fkey FOREIGN KEY (source_block_group_id) REFERENCES public.block_groups(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.edges ADD CONSTRAINT edges_target_block_group_id_fkey FOREIGN KEY (target_block_group_id) REFERENCES public.block_groups(id) ON DELETE CASCADE;

-- Block Groups
ALTER TABLE ONLY public.block_groups ADD CONSTRAINT block_groups_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.block_groups ADD CONSTRAINT block_groups_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.block_groups ADD CONSTRAINT block_groups_parent_group_id_fkey FOREIGN KEY (parent_group_id) REFERENCES public.block_groups(id) ON DELETE CASCADE;

-- Block Definitions
ALTER TABLE ONLY public.block_definitions ADD CONSTRAINT block_definitions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.block_definitions ADD CONSTRAINT block_definitions_parent_block_id_fkey FOREIGN KEY (parent_block_id) REFERENCES public.block_definitions(id) ON DELETE SET NULL;

-- Block Versions
ALTER TABLE ONLY public.block_versions ADD CONSTRAINT block_versions_block_id_fkey FOREIGN KEY (block_id) REFERENCES public.block_definitions(id) ON DELETE CASCADE;

-- Runs
ALTER TABLE ONLY public.runs ADD CONSTRAINT runs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.runs ADD CONSTRAINT runs_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id);
ALTER TABLE ONLY public.runs ADD CONSTRAINT runs_start_step_id_fkey FOREIGN KEY (start_step_id) REFERENCES public.steps(id);
ALTER TABLE ONLY public.runs ADD CONSTRAINT runs_triggered_by_user_fkey FOREIGN KEY (triggered_by_user) REFERENCES public.users(id);

-- Run Number Sequences
ALTER TABLE ONLY public.run_number_sequences ADD CONSTRAINT run_number_sequences_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;

-- Step Runs
ALTER TABLE ONLY public.step_runs ADD CONSTRAINT step_runs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.step_runs ADD CONSTRAINT step_runs_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.runs(id) ON DELETE CASCADE;

-- Schedules
ALTER TABLE ONLY public.schedules ADD CONSTRAINT schedules_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.schedules ADD CONSTRAINT schedules_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.schedules ADD CONSTRAINT schedules_start_step_id_fkey FOREIGN KEY (start_step_id) REFERENCES public.steps(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.schedules ADD CONSTRAINT schedules_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);
ALTER TABLE ONLY public.schedules ADD CONSTRAINT schedules_last_run_id_fkey FOREIGN KEY (last_run_id) REFERENCES public.runs(id);

-- Credentials
ALTER TABLE ONLY public.credentials ADD CONSTRAINT credentials_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- Secrets
ALTER TABLE ONLY public.secrets ADD CONSTRAINT secrets_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.secrets ADD CONSTRAINT secrets_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

-- Usage
ALTER TABLE ONLY public.usage_records ADD CONSTRAINT usage_records_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.usage_records ADD CONSTRAINT usage_records_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id);
ALTER TABLE ONLY public.usage_records ADD CONSTRAINT usage_records_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.runs(id);
ALTER TABLE ONLY public.usage_records ADD CONSTRAINT usage_records_step_run_id_fkey FOREIGN KEY (step_run_id) REFERENCES public.step_runs(id);

ALTER TABLE ONLY public.usage_daily_aggregates ADD CONSTRAINT usage_daily_aggregates_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.usage_daily_aggregates ADD CONSTRAINT usage_daily_aggregates_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id);

ALTER TABLE ONLY public.usage_budgets ADD CONSTRAINT usage_budgets_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.usage_budgets ADD CONSTRAINT usage_budgets_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id);

-- Audit
ALTER TABLE ONLY public.audit_logs ADD CONSTRAINT audit_logs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);

-- Copilot
ALTER TABLE ONLY public.copilot_sessions ADD CONSTRAINT copilot_sessions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.copilot_sessions ADD CONSTRAINT copilot_sessions_context_project_id_fkey FOREIGN KEY (context_project_id) REFERENCES public.projects(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.copilot_sessions ADD CONSTRAINT copilot_sessions_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE SET NULL;
ALTER TABLE ONLY public.copilot_messages ADD CONSTRAINT copilot_messages_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.copilot_sessions(id) ON DELETE CASCADE;

-- ============================================================================
-- RAG (Retrieval-Augmented Generation) Tables
-- ============================================================================

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

COMMENT ON TABLE public.vector_collections IS 'RAG vector collections with tenant isolation';
COMMENT ON COLUMN public.vector_collections.dimension IS 'Vector dimension (1536 for text-embedding-3-small, 3072 for text-embedding-3-large). Note: vector_documents.embedding is fixed at 1536d - use separate collections for different dimensions.';

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
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

COMMENT ON TABLE public.vector_documents IS 'RAG vector documents with embeddings, tenant-isolated. Note: embedding column is fixed at 1536 dimensions (OpenAI text-embedding-3-small). For other dimensions, create separate tables or use dynamic column types in future versions.';
COMMENT ON COLUMN public.vector_documents.content IS 'Document content (LangChain Document.page_content equivalent)';
COMMENT ON COLUMN public.vector_documents.metadata IS 'Document metadata (LangChain Document.metadata equivalent)';
COMMENT ON COLUMN public.vector_documents.embedding IS 'Vector embedding from embedding model';
COMMENT ON COLUMN public.vector_documents.source_type IS 'Source type: url, file, text, api';

-- Vector Collections Constraints
ALTER TABLE ONLY public.vector_collections ADD CONSTRAINT vector_collections_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.vector_collections ADD CONSTRAINT unique_collection_per_tenant UNIQUE (tenant_id, name);

-- Vector Documents Constraints
ALTER TABLE ONLY public.vector_documents ADD CONSTRAINT vector_documents_pkey PRIMARY KEY (id);

-- Vector Indexes
CREATE INDEX idx_vector_collections_tenant ON public.vector_collections USING btree (tenant_id);
CREATE INDEX idx_vector_documents_tenant_collection ON public.vector_documents USING btree (tenant_id, collection_id);
CREATE INDEX idx_vector_documents_embedding ON public.vector_documents USING ivfflat (embedding public.vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_vector_documents_metadata ON public.vector_documents USING gin (metadata);

-- Vector Triggers
CREATE OR REPLACE FUNCTION public.update_vector_collections_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_vector_collections_updated_at BEFORE UPDATE ON public.vector_collections FOR EACH ROW EXECUTE FUNCTION public.update_vector_collections_updated_at();

CREATE OR REPLACE FUNCTION public.update_vector_documents_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_vector_documents_updated_at BEFORE UPDATE ON public.vector_documents FOR EACH ROW EXECUTE FUNCTION public.update_vector_documents_updated_at();

-- Vector Foreign Keys
ALTER TABLE ONLY public.vector_collections ADD CONSTRAINT vector_collections_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.vector_documents ADD CONSTRAINT vector_documents_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.vector_documents ADD CONSTRAINT vector_documents_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES public.vector_collections(id) ON DELETE CASCADE;

-- ============================================================================
-- Agent Execution Memory (N8N-style Agent Memory for ReAct Loop)
-- ============================================================================

--
-- Name: agent_memory; Type: TABLE; Schema: public; Owner: -
-- Stores conversation history for agent block executions
--

CREATE TABLE public.agent_memory (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    run_id uuid NOT NULL,
    step_id uuid NOT NULL,
    role character varying(50) NOT NULL,
    content text NOT NULL,
    tool_calls jsonb,
    tool_call_id character varying(100),
    metadata jsonb DEFAULT '{}'::jsonb,
    sequence_number integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT agent_memory_role_check CHECK (((role)::text = ANY ((ARRAY['user'::character varying, 'assistant'::character varying, 'system'::character varying, 'tool'::character varying])::text[])))
);

COMMENT ON TABLE public.agent_memory IS 'Agent execution memory for ReAct loop conversation history';
COMMENT ON COLUMN public.agent_memory.role IS 'Message role: user, assistant, system, tool';
COMMENT ON COLUMN public.agent_memory.tool_calls IS 'Tool calls made by assistant (JSON array of tool call objects)';
COMMENT ON COLUMN public.agent_memory.tool_call_id IS 'ID of tool call this message responds to (for role=tool)';
COMMENT ON COLUMN public.agent_memory.sequence_number IS 'Order of message within the run+step conversation';

-- Agent Memory Constraints
ALTER TABLE ONLY public.agent_memory ADD CONSTRAINT agent_memory_pkey PRIMARY KEY (id);

-- Agent Memory Indexes
CREATE INDEX idx_agent_memory_run_step ON public.agent_memory USING btree (run_id, step_id);
CREATE INDEX idx_agent_memory_sequence ON public.agent_memory USING btree (run_id, step_id, sequence_number);

-- Agent Memory Foreign Keys
ALTER TABLE ONLY public.agent_memory ADD CONSTRAINT agent_memory_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.agent_memory ADD CONSTRAINT agent_memory_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.runs(id) ON DELETE CASCADE;

-- ============================================================================
-- Error Workflow Configuration
-- ============================================================================

-- Projects: Error Workflow
ALTER TABLE public.projects ADD COLUMN error_workflow_id uuid;
ALTER TABLE public.projects ADD COLUMN error_workflow_config jsonb DEFAULT '{}'::jsonb;
ALTER TABLE ONLY public.projects ADD CONSTRAINT projects_error_workflow_id_fkey FOREIGN KEY (error_workflow_id) REFERENCES public.projects(id) ON DELETE SET NULL;

COMMENT ON COLUMN public.projects.error_workflow_id IS 'Project to execute when this project run fails';
COMMENT ON COLUMN public.projects.error_workflow_config IS 'Error workflow config: {"trigger_on": ["failed"], "input_mapping": {...}}';

-- Runs: Error Trigger Source
ALTER TABLE public.runs ADD COLUMN parent_run_id uuid;
ALTER TABLE public.runs ADD COLUMN error_trigger_source jsonb;
ALTER TABLE ONLY public.runs ADD CONSTRAINT runs_parent_run_id_fkey FOREIGN KEY (parent_run_id) REFERENCES public.runs(id) ON DELETE SET NULL;
CREATE INDEX idx_runs_parent ON public.runs USING btree (parent_run_id) WHERE (parent_run_id IS NOT NULL);

COMMENT ON COLUMN public.runs.parent_run_id IS 'Parent run that triggered this error workflow run';
COMMENT ON COLUMN public.runs.error_trigger_source IS 'Error info: {"original_run_id", "error_step_id", "error_step_name", "error_message"}';

-- ============================================================================
-- Phase 2: Debug & Retry Features
-- ============================================================================

-- Steps: Retry Configuration
ALTER TABLE public.steps ADD COLUMN retry_config jsonb DEFAULT '{}'::jsonb;
COMMENT ON COLUMN public.steps.retry_config IS 'Retry config: {"max_retries": 3, "delay_ms": 1000, "exponential_backoff": true, "retry_on_errors": ["TIMEOUT"]}';

-- StepRuns: Pinned Input for Debugging
ALTER TABLE public.step_runs ADD COLUMN pinned_input jsonb;
COMMENT ON COLUMN public.step_runs.pinned_input IS 'Pinned input data for debugging/replay';

-- Agent Chat Sessions (for agent-chat trigger)
CREATE TABLE public.agent_chat_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    start_step_id uuid NOT NULL,
    user_id character varying(255) NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT agent_chat_sessions_status_check CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'closed'::character varying])::text[])))
);

COMMENT ON TABLE public.agent_chat_sessions IS 'Chat sessions for agent-chat trigger type';

-- Agent Chat Sessions Constraints
ALTER TABLE ONLY public.agent_chat_sessions ADD CONSTRAINT agent_chat_sessions_pkey PRIMARY KEY (id);

-- Agent Chat Sessions Indexes
CREATE INDEX idx_agent_chat_sessions_tenant ON public.agent_chat_sessions USING btree (tenant_id);
CREATE INDEX idx_agent_chat_sessions_project ON public.agent_chat_sessions USING btree (project_id);
CREATE INDEX idx_agent_chat_sessions_user ON public.agent_chat_sessions USING btree (tenant_id, user_id);
CREATE INDEX idx_agent_chat_sessions_active ON public.agent_chat_sessions USING btree (tenant_id, user_id, status) WHERE ((status)::text = 'active'::text);

-- Agent Chat Sessions Foreign Keys
ALTER TABLE ONLY public.agent_chat_sessions ADD CONSTRAINT agent_chat_sessions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.agent_chat_sessions ADD CONSTRAINT agent_chat_sessions_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.agent_chat_sessions ADD CONSTRAINT agent_chat_sessions_start_step_id_fkey FOREIGN KEY (start_step_id) REFERENCES public.steps(id) ON DELETE CASCADE;

-- Agent Chat Sessions Trigger
CREATE TRIGGER trigger_agent_chat_sessions_updated_at BEFORE UPDATE ON public.agent_chat_sessions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- ============================================================================
-- Phase 3: Templates & Streaming
-- ============================================================================

-- Project Templates
CREATE TABLE public.project_templates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid,
    name character varying(255) NOT NULL,
    description text,
    category character varying(100),
    tags jsonb DEFAULT '[]'::jsonb,
    definition jsonb NOT NULL,
    variables jsonb DEFAULT '{}'::jsonb,
    thumbnail_url text,
    author_name character varying(255),
    download_count integer DEFAULT 0,
    is_featured boolean DEFAULT false,
    -- Phase 4: Marketplace fields
    visibility character varying(50) DEFAULT 'private'::character varying NOT NULL,
    review_status character varying(50),
    price_usd numeric(10,2) DEFAULT 0,
    rating numeric(3,2),
    review_count integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT project_templates_visibility_check CHECK (((visibility)::text = ANY ((ARRAY['private'::character varying, 'tenant'::character varying, 'public'::character varying])::text[]))),
    CONSTRAINT project_templates_review_status_check CHECK ((review_status IS NULL OR (review_status)::text = ANY ((ARRAY['pending'::character varying, 'approved'::character varying, 'rejected'::character varying])::text[])))
);

COMMENT ON TABLE public.project_templates IS 'Reusable workflow templates';
COMMENT ON COLUMN public.project_templates.tenant_id IS 'NULL for system templates';
COMMENT ON COLUMN public.project_templates.definition IS 'Snapshot of steps, edges, block_groups';
COMMENT ON COLUMN public.project_templates.variables IS 'Template variables for customization';
COMMENT ON COLUMN public.project_templates.visibility IS 'private (owner only), tenant (organization), public (marketplace)';

-- Project Templates Constraints
ALTER TABLE ONLY public.project_templates ADD CONSTRAINT project_templates_pkey PRIMARY KEY (id);

-- Project Templates Indexes
CREATE INDEX idx_project_templates_tenant ON public.project_templates USING btree (tenant_id);
CREATE INDEX idx_project_templates_category ON public.project_templates USING btree (category);
CREATE INDEX idx_project_templates_visibility ON public.project_templates USING btree (visibility);
CREATE INDEX idx_project_templates_featured ON public.project_templates USING btree (is_featured) WHERE (is_featured = true);
CREATE INDEX idx_project_templates_public ON public.project_templates USING btree (visibility, review_status) WHERE ((visibility)::text = 'public'::text AND (review_status)::text = 'approved'::text);

-- Project Templates Foreign Keys
ALTER TABLE ONLY public.project_templates ADD CONSTRAINT project_templates_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

-- Project Templates Trigger
CREATE TRIGGER trigger_project_templates_updated_at BEFORE UPDATE ON public.project_templates FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- StepRuns: Streaming Output
ALTER TABLE public.step_runs ADD COLUMN streaming_output jsonb DEFAULT '[]'::jsonb;
COMMENT ON COLUMN public.step_runs.streaming_output IS 'Streaming output chunks: [{"chunk": "...", "timestamp": "...", "type": "text|json"}]';

-- ============================================================================
-- Phase 4: Marketplace & Git Integration
-- ============================================================================

-- Template Reviews
CREATE TABLE public.template_reviews (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    template_id uuid NOT NULL,
    user_id uuid NOT NULL,
    rating integer NOT NULL,
    comment text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT template_reviews_rating_check CHECK ((rating >= 1 AND rating <= 5))
);

COMMENT ON TABLE public.template_reviews IS 'User reviews for marketplace templates';

-- Template Reviews Constraints
ALTER TABLE ONLY public.template_reviews ADD CONSTRAINT template_reviews_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.template_reviews ADD CONSTRAINT template_reviews_unique_user UNIQUE (template_id, user_id);

-- Template Reviews Indexes
CREATE INDEX idx_template_reviews_template ON public.template_reviews USING btree (template_id);

-- Template Reviews Foreign Keys
ALTER TABLE ONLY public.template_reviews ADD CONSTRAINT template_reviews_template_id_fkey FOREIGN KEY (template_id) REFERENCES public.project_templates(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.template_reviews ADD CONSTRAINT template_reviews_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);

-- Project Git Sync
CREATE TABLE public.project_git_sync (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    repository_url text NOT NULL,
    branch character varying(255) DEFAULT 'main'::character varying NOT NULL,
    file_path character varying(500) DEFAULT 'workflow.json'::character varying NOT NULL,
    sync_direction character varying(50) DEFAULT 'bidirectional'::character varying NOT NULL,
    auto_sync boolean DEFAULT false NOT NULL,
    last_sync_at timestamp with time zone,
    last_commit_sha character varying(100),
    credentials_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT project_git_sync_direction_check CHECK (((sync_direction)::text = ANY ((ARRAY['push'::character varying, 'pull'::character varying, 'bidirectional'::character varying])::text[])))
);

COMMENT ON TABLE public.project_git_sync IS 'Git repository sync configuration for projects';
COMMENT ON COLUMN public.project_git_sync.sync_direction IS 'push (project->git), pull (git->project), bidirectional';

-- Project Git Sync Constraints
ALTER TABLE ONLY public.project_git_sync ADD CONSTRAINT project_git_sync_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.project_git_sync ADD CONSTRAINT project_git_sync_project_unique UNIQUE (project_id);

-- Project Git Sync Indexes
CREATE INDEX idx_project_git_sync_tenant ON public.project_git_sync USING btree (tenant_id);
CREATE INDEX idx_project_git_sync_auto ON public.project_git_sync USING btree (auto_sync) WHERE (auto_sync = true);

-- Project Git Sync Foreign Keys
ALTER TABLE ONLY public.project_git_sync ADD CONSTRAINT project_git_sync_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id);
ALTER TABLE ONLY public.project_git_sync ADD CONSTRAINT project_git_sync_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.project_git_sync ADD CONSTRAINT project_git_sync_credentials_id_fkey FOREIGN KEY (credentials_id) REFERENCES public.credentials(id) ON DELETE SET NULL;

-- Project Git Sync Trigger
CREATE TRIGGER trigger_project_git_sync_updated_at BEFORE UPDATE ON public.project_git_sync FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- Custom Block Packages
CREATE TABLE public.custom_block_packages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    version character varying(50) NOT NULL,
    description text,
    bundle_url text,
    blocks jsonb NOT NULL,
    dependencies jsonb DEFAULT '[]'::jsonb,
    status character varying(50) DEFAULT 'draft'::character varying NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT custom_block_packages_status_check CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'published'::character varying, 'deprecated'::character varying])::text[])))
);

COMMENT ON TABLE public.custom_block_packages IS 'Custom block SDK packages';
COMMENT ON COLUMN public.custom_block_packages.blocks IS 'Array of block definitions in this package';
COMMENT ON COLUMN public.custom_block_packages.dependencies IS 'NPM-style dependencies';

-- Custom Block Packages Constraints
ALTER TABLE ONLY public.custom_block_packages ADD CONSTRAINT custom_block_packages_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.custom_block_packages ADD CONSTRAINT custom_block_packages_version_unique UNIQUE (tenant_id, name, version);

-- Custom Block Packages Indexes
CREATE INDEX idx_custom_block_packages_tenant ON public.custom_block_packages USING btree (tenant_id);
CREATE INDEX idx_custom_block_packages_status ON public.custom_block_packages USING btree (status);

-- Custom Block Packages Foreign Keys
ALTER TABLE ONLY public.custom_block_packages ADD CONSTRAINT custom_block_packages_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.custom_block_packages ADD CONSTRAINT custom_block_packages_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);

-- Custom Block Packages Trigger
CREATE TRIGGER trigger_custom_block_packages_updated_at BEFORE UPDATE ON public.custom_block_packages FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

--
-- PostgreSQL database dump complete
--
