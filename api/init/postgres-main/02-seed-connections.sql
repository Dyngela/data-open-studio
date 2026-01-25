-- Seed Database Connections
-- Pre-configured connections to the test databases

-- Clear existing seed data (optional, comment out if you want to preserve data)
-- TRUNCATE metadata_database RESTART IDENTITY CASCADE;
-- TRUNCATE metadata_sftp RESTART IDENTITY CASCADE;

-- Insert DB connections only if table is empty (first run)
INSERT INTO metadata_database (host, port, "user", password, database_name, ssl_mode, extra)
SELECT * FROM (VALUES
    -- PostgreSQL Test Database
    ('postgres-test', '5432', 'testuser', 'testpass', 'testdb', 'disable', '{"description": "PostgreSQL Test Database with sample data"}'),

    -- SQL Server Test Database
    ('sqlserver', '1433', 'sa', 'TestPass123!', 'testdb', 'disable', '{"description": "SQL Server Test Database with sample data", "driver": "sqlserver"}')
) AS seed(host, port, "user", password, database_name, ssl_mode, extra)
WHERE NOT EXISTS (SELECT 1 FROM metadata_database LIMIT 1);

-- Insert SFTP connections only if table is empty (first run)
INSERT INTO metadata_sftp (host, port, "user", password, private_key, base_path, extra)
SELECT * FROM (VALUES
    -- Example SFTP connection (localhost for dev)
    ('localhost', '22', 'sftpuser', 'sftppass', '', '/data', '{"description": "Local SFTP for development"}'),

    -- Example SFTP with key auth
    ('sftp.example.com', '22', 'keyuser', '', '', '/uploads', '{"description": "Example SFTP with key authentication"}')
) AS seed(host, port, "user", password, private_key, base_path, extra)
WHERE NOT EXISTS (SELECT 1 FROM metadata_sftp LIMIT 1);

-- Display seeded data
DO $$
BEGIN
    RAISE NOTICE 'Database connections seeded:';
END $$;

SELECT id, host, port, database_name, ssl_mode FROM metadata_database;
SELECT id, host, port, "user", base_path FROM metadata_sftp;
