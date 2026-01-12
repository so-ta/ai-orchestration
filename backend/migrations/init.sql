-- AI Orchestration - PostgreSQL Initialization
-- This file is executed on first PostgreSQL startup
-- Schema is managed by sqldef (psqldef)
--
-- To apply schema: make db-apply
-- To load seed data: make db-seed

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
