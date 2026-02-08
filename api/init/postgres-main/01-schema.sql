-- Main Database Schema
-- Creates tables for metadata connections if they don't exist
-- Note: The Go app also runs auto-migrate, but this ensures tables exist for seeding

-- DB Metadata table (connection info for databases)
CREATE TABLE IF NOT EXISTS metadata_database (
    id SERIAL PRIMARY KEY,
    host VARCHAR(255) NOT NULL,
    port int NOT NULL,
    "user" VARCHAR(100) NOT NULL,
    password VARCHAR(255) NOT NULL,
    database_name VARCHAR(100) NOT NULL,
    ssl_mode VARCHAR(50) DEFAULT 'disable',
    extra TEXT DEFAULT ''
);

-- SFTP Metadata table (connection info for SFTP servers)
CREATE TABLE IF NOT EXISTS metadata_sftp (
    id SERIAL PRIMARY KEY,
    host VARCHAR(255) NOT NULL,
    port int NOT NULL DEFAULT 22,
    "user" VARCHAR(100) NOT NULL,
    password VARCHAR(255) DEFAULT '',
    private_key TEXT DEFAULT '',
    base_path VARCHAR(500) DEFAULT '/',
    extra TEXT DEFAULT ''
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_metadata_database_host ON metadata_database(host);
CREATE INDEX IF NOT EXISTS idx_metadata_sftp_host ON metadata_sftp(host);
