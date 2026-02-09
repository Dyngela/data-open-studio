-- Main Database Schema
-- Creates all tables matching GORM models in api/internal/api/models/
-- Note: The Go app also runs auto-migrate in dev mode, but this ensures tables exist for seeding

-- ============================================================
-- Users
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(256) UNIQUE NOT NULL,
    password TEXT NOT NULL,
    prenom TEXT NOT NULL,
    nom TEXT NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    actif BOOLEAN DEFAULT true,
    refresh_token TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- ============================================================
-- Jobs
-- ============================================================
CREATE TABLE IF NOT EXISTS job (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    description TEXT DEFAULT '',
    file_path TEXT DEFAULT '',
    creator_id BIGINT,
    active BOOLEAN DEFAULT false,
    visibility TEXT DEFAULT 'private',
    output_path TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- ============================================================
-- Job User Access (many-to-many junction)
-- ============================================================
CREATE TABLE IF NOT EXISTS job_user_access (
    job_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role TEXT DEFAULT 'viewer',
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (job_id, user_id)
);

-- ============================================================
-- Nodes
-- ============================================================
CREATE TABLE IF NOT EXISTS node (
    id SERIAL PRIMARY KEY,
    type TEXT NOT NULL DEFAULT '',
    name TEXT DEFAULT '',
    xpos REAL DEFAULT 0,
    ypos REAL DEFAULT 0,
    data JSONB,
    job_id BIGINT,
    CONSTRAINT fk_node_job FOREIGN KEY (job_id) REFERENCES job(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_node_job_id ON node(job_id);

-- ============================================================
-- Ports (connections between nodes)
-- ============================================================
CREATE TABLE IF NOT EXISTS port (
    id SERIAL PRIMARY KEY,
    type TEXT NOT NULL DEFAULT '',
    node_id BIGINT,
    connected_node_id BIGINT,
    CONSTRAINT fk_port_node FOREIGN KEY (node_id) REFERENCES node(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_port_node_id ON port(node_id);

-- ============================================================
-- Metadata Database (connection info for databases)
-- ============================================================
CREATE TABLE IF NOT EXISTS metadata_database (
    id SERIAL PRIMARY KEY,
    host TEXT NOT NULL DEFAULT '',
    port BIGINT DEFAULT 5432,
    "user" TEXT NOT NULL DEFAULT '',
    password TEXT DEFAULT '',
    database_name TEXT NOT NULL DEFAULT '',
    ssl_mode TEXT DEFAULT 'disable',
    extra TEXT DEFAULT '',
    db_type VARCHAR(20) DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_metadata_database_host ON metadata_database(host);

-- ============================================================
-- Metadata SFTP (connection info for SFTP servers)
-- ============================================================
CREATE TABLE IF NOT EXISTS metadata_sftp (
    id SERIAL PRIMARY KEY,
    host TEXT NOT NULL DEFAULT '',
    port BIGINT DEFAULT 22,
    "user" TEXT NOT NULL DEFAULT '',
    password TEXT DEFAULT '',
    private_key TEXT DEFAULT '',
    base_path TEXT DEFAULT '/',
    extra TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_metadata_sftp_host ON metadata_sftp(host);

-- ============================================================
-- Triggers
-- ============================================================
CREATE TABLE IF NOT EXISTS trigger (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'paused',
    creator_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    polling_interval INTEGER DEFAULT 60,
    last_polled_at TIMESTAMPTZ,
    last_error TEXT DEFAULT '',
    config JSONB,
    CONSTRAINT fk_trigger_creator FOREIGN KEY (creator_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_trigger_deleted_at ON trigger(deleted_at);

-- ============================================================
-- Trigger Rules
-- ============================================================
CREATE TABLE IF NOT EXISTS trigger_rule (
    id SERIAL PRIMARY KEY,
    trigger_id BIGINT NOT NULL,
    name TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    conditions JSONB,
    CONSTRAINT fk_trigger_rule_trigger FOREIGN KEY (trigger_id) REFERENCES trigger(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_trigger_rule_deleted_at ON trigger_rule(deleted_at);

-- ============================================================
-- Trigger Jobs (links triggers to jobs)
-- ============================================================
CREATE TABLE IF NOT EXISTS trigger_job (
    id SERIAL PRIMARY KEY,
    trigger_id BIGINT NOT NULL,
    job_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    priority INTEGER DEFAULT 0,
    active BOOLEAN DEFAULT true,
    pass_event_data BOOLEAN DEFAULT false,
    CONSTRAINT fk_trigger_job_trigger FOREIGN KEY (trigger_id) REFERENCES trigger(id) ON DELETE CASCADE,
    CONSTRAINT fk_trigger_job_job FOREIGN KEY (job_id) REFERENCES job(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_trigger_job_deleted_at ON trigger_job(deleted_at);

-- ============================================================
-- Trigger Executions (audit log for trigger runs)
-- ============================================================
CREATE TABLE IF NOT EXISTS trigger_execution (
    id SERIAL PRIMARY KEY,
    trigger_id BIGINT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT '',
    event_count INTEGER DEFAULT 0,
    jobs_triggered INTEGER DEFAULT 0,
    error TEXT DEFAULT '',
    event_sample JSONB
);

CREATE INDEX IF NOT EXISTS idx_trigger_execution_trigger_id ON trigger_execution(trigger_id);
