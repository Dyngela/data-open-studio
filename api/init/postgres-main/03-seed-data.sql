-- Seed Data for all application tables
-- Populates users, jobs, nodes, ports, triggers with sample data

-- ============================================================
-- Users
-- ============================================================
-- Passwords are bcrypt hashes of "password123"
INSERT INTO users (email, password, prenom, nom, role, actif, created_at, updated_at)
SELECT * FROM (VALUES
    ('admin@opendata.io'::VARCHAR(256),
     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'::TEXT,
     'Alice'::TEXT, 'Martin'::TEXT, 'admin'::VARCHAR(50), true::BOOLEAN,
     now()::TIMESTAMPTZ, now()::TIMESTAMPTZ),

    ('bob@opendata.io'::VARCHAR(256),
     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'::TEXT,
     'Bob'::TEXT, 'Dupont'::TEXT, 'user'::VARCHAR(50), true::BOOLEAN,
     now()::TIMESTAMPTZ, now()::TIMESTAMPTZ),

    ('claire@opendata.io'::VARCHAR(256),
     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'::TEXT,
     'Claire'::TEXT, 'Bernard'::TEXT, 'user'::VARCHAR(50), true::BOOLEAN,
     now()::TIMESTAMPTZ, now()::TIMESTAMPTZ)
) AS seed(email, password, prenom, nom, role, actif, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM users LIMIT 1);

-- ============================================================
-- Jobs
-- ============================================================
INSERT INTO job (name, description, file_path, creator_id, active, visibility, output_path, created_at, updated_at)
SELECT * FROM (VALUES
    -- Job 1: Simple ETL pipeline
    ('Customer ETL'::TEXT,
     'Extract customers from PostgreSQL, transform names, load into target DB'::TEXT,
     '/projects/etl/'::TEXT, 1::BIGINT, true::BOOLEAN, 'public'::TEXT, '/output/customers/'::TEXT,
     now()::TIMESTAMPTZ, now()::TIMESTAMPTZ),

    -- Job 2: Daily sales report
    ('Daily Sales Report'::TEXT,
     'Aggregate daily sales data and output summary to reporting database'::TEXT,
     '/projects/reports/'::TEXT, 1::BIGINT, true::BOOLEAN, 'private'::TEXT, '/output/sales/'::TEXT,
     now()::TIMESTAMPTZ, now()::TIMESTAMPTZ),

    -- Job 3: Data quality check
    ('Data Quality Check'::TEXT,
     'Run validation checks on incoming order data and log anomalies'::TEXT,
     '/projects/quality/'::TEXT, 2::BIGINT, false::BOOLEAN, 'private'::TEXT, '/output/quality/'::TEXT,
     now()::TIMESTAMPTZ, now()::TIMESTAMPTZ)
) AS seed(name, description, file_path, creator_id, active, visibility, output_path, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM job LIMIT 1);

-- ============================================================
-- Job User Access (sharing)
-- ============================================================
INSERT INTO job_user_access (job_id, user_id, role, created_at)
SELECT * FROM (VALUES
    -- Bob has editor access to Customer ETL
    (1::BIGINT, 2::BIGINT, 'editor'::TEXT, now()::TIMESTAMPTZ),
    -- Claire has viewer access to Customer ETL
    (1::BIGINT, 3::BIGINT, 'viewer'::TEXT, now()::TIMESTAMPTZ),
    -- Claire has editor access to Daily Sales Report
    (2::BIGINT, 3::BIGINT, 'editor'::TEXT, now()::TIMESTAMPTZ)
) AS seed(job_id, user_id, role, created_at)
WHERE NOT EXISTS (SELECT 1 FROM job_user_access LIMIT 1);

-- ============================================================
-- Nodes for Job 1 (Customer ETL): start -> db_input -> map -> db_output
-- ============================================================
INSERT INTO node (id, type, name, xpos, ypos, data, job_id)
SELECT * FROM (VALUES
    -- Start node
    (1, 'start'::TEXT, 'Start'::TEXT, 100::REAL, 200::REAL, NULL::JSONB, 1::BIGINT),

    -- DB Input: read customers from source
    (2, 'db_input'::TEXT, 'Read Customers'::TEXT, 300::REAL, 200::REAL,
     '{"query":"SELECT id, first_name, last_name, email, created_at FROM customers","dbschema":"public","batchSize":500,"connection":{"type":"postgres","host":"postgres-test","port":5432,"database":"testdb","username":"testuser","password":"testpass","sslMode":"disable"},"dataModels":[{"name":"id","type":"integer","goType":"int64","nullable":false},{"name":"first_name","type":"text","goType":"string","nullable":false},{"name":"last_name","type":"text","goType":"string","nullable":false},{"name":"email","type":"text","goType":"string","nullable":true},{"name":"created_at","type":"timestamp","goType":"time.Time","nullable":false}]}'::JSONB,
     1::BIGINT),

    -- Map: transform names to uppercase
    (3, 'map'::TEXT, 'Transform Names'::TEXT, 500::REAL, 200::REAL,
     '{"inputs":[{"name":"A","portId":1,"schema":[{"name":"id","type":"integer","goType":"int64","nullable":false},{"name":"first_name","type":"text","goType":"string","nullable":false},{"name":"last_name","type":"text","goType":"string","nullable":false},{"name":"email","type":"text","goType":"string","nullable":true},{"name":"created_at","type":"timestamp","goType":"time.Time","nullable":false}]}],"outputs":[{"name":"main","portId":2,"columns":[{"name":"id","dataType":"int64","funcType":"direct","inputRef":"A.id"},{"name":"full_name","dataType":"string","funcType":"library","libFunc":"Concat","args":[{"type":"column","value":"A.first_name"},{"type":"literal","value":" "},{"type":"column","value":"A.last_name"}]},{"name":"email","dataType":"string","funcType":"direct","inputRef":"A.email"},{"name":"created_at","dataType":"time.Time","funcType":"direct","inputRef":"A.created_at"}]}]}'::JSONB,
     1::BIGINT),

    -- DB Output: write to target table
    (4, 'db_output'::TEXT, 'Write Customers'::TEXT, 700::REAL, 200::REAL,
     '{"table":"customers_clean","mode":"insert","batchSize":500,"dbschema":"public","connection":{"type":"postgres","host":"postgres-test","port":5432,"database":"testdb","username":"testuser","password":"testpass","sslMode":"disable"},"dataModel":[{"name":"id","type":"integer","goType":"int64","nullable":false},{"name":"full_name","type":"text","goType":"string","nullable":false},{"name":"email","type":"text","goType":"string","nullable":true},{"name":"created_at","type":"timestamp","goType":"time.Time","nullable":false}]}'::JSONB,
     1::BIGINT)
) AS seed(id, type, name, xpos, ypos, data, job_id)
WHERE NOT EXISTS (SELECT 1 FROM node LIMIT 1);

-- Reset the node sequence to the max id
SELECT setval('node_id_seq', (SELECT COALESCE(MAX(id), 1) FROM node));

-- ============================================================
-- Nodes for Job 2 (Daily Sales Report): start -> db_input -> log
-- ============================================================
INSERT INTO node (id, type, name, xpos, ypos, data, job_id)
SELECT * FROM (VALUES
    (5, 'start'::TEXT, 'Start'::TEXT, 100::REAL, 200::REAL, NULL::JSONB, 2::BIGINT),

    (6, 'db_input'::TEXT, 'Read Sales'::TEXT, 300::REAL, 200::REAL,
     '{"query":"SELECT product_name, SUM(quantity) as total_qty, SUM(amount) as total_amount FROM sales WHERE sale_date = CURRENT_DATE GROUP BY product_name","dbschema":"public","batchSize":1000,"connection":{"type":"postgres","host":"postgres-test","port":5432,"database":"testdb","username":"testuser","password":"testpass","sslMode":"disable"},"dataModels":[{"name":"product_name","type":"text","goType":"string","nullable":false},{"name":"total_qty","type":"bigint","goType":"int64","nullable":true},{"name":"total_amount","type":"numeric","goType":"float64","nullable":true}]}'::JSONB,
     2::BIGINT),

    (7, 'log'::TEXT, 'Log Sales Summary'::TEXT, 500::REAL, 200::REAL, NULL::JSONB, 2::BIGINT)
) AS seed(id, type, name, xpos, ypos, data, job_id)
WHERE NOT EXISTS (SELECT 1 FROM node WHERE job_id = 2 LIMIT 1);

SELECT setval('node_id_seq', (SELECT COALESCE(MAX(id), 1) FROM node));

-- ============================================================
-- Ports (connections between nodes)
-- ============================================================
INSERT INTO port (type, node_id, connected_node_id)
SELECT * FROM (VALUES
    -- Job 1 flow: start -> db_input -> map -> db_output
    ('node_flow_output'::TEXT, 1::BIGINT, 2::BIGINT),  -- start out -> db_input
    ('node_flow_input'::TEXT,  2::BIGINT, 1::BIGINT),   -- db_input in <- start
    ('output'::TEXT,           2::BIGINT, 3::BIGINT),    -- db_input data out -> map
    ('input'::TEXT,            3::BIGINT, 2::BIGINT),    -- map data in <- db_input
    ('node_flow_output'::TEXT, 2::BIGINT, 3::BIGINT),   -- db_input flow -> map
    ('node_flow_input'::TEXT,  3::BIGINT, 2::BIGINT),   -- map flow <- db_input
    ('output'::TEXT,           3::BIGINT, 4::BIGINT),    -- map data out -> db_output
    ('input'::TEXT,            4::BIGINT, 3::BIGINT),    -- db_output data in <- map
    ('node_flow_output'::TEXT, 3::BIGINT, 4::BIGINT),   -- map flow -> db_output
    ('node_flow_input'::TEXT,  4::BIGINT, 3::BIGINT),   -- db_output flow <- map

    -- Job 2 flow: start -> db_input -> log
    ('node_flow_output'::TEXT, 5::BIGINT, 6::BIGINT),   -- start out -> db_input
    ('node_flow_input'::TEXT,  6::BIGINT, 5::BIGINT),   -- db_input in <- start
    ('node_flow_output'::TEXT, 6::BIGINT, 7::BIGINT),   -- db_input flow -> log
    ('node_flow_input'::TEXT,  7::BIGINT, 6::BIGINT),   -- log flow <- db_input
    ('output'::TEXT,           6::BIGINT, 7::BIGINT),    -- db_input data out -> log
    ('input'::TEXT,            7::BIGINT, 6::BIGINT)     -- log data in <- db_input
) AS seed(type, node_id, connected_node_id)
WHERE NOT EXISTS (SELECT 1 FROM port LIMIT 1);

-- ============================================================
-- Triggers
-- ============================================================
INSERT INTO trigger (name, description, type, status, creator_id, polling_interval, config, created_at, updated_at)
SELECT * FROM (VALUES
    -- Database polling trigger: watch for new orders
    ('New Orders Watcher'::TEXT,
     'Polls the orders table for new rows and triggers the Customer ETL job'::TEXT,
     'database'::VARCHAR(20), 'active'::VARCHAR(20), 1::BIGINT, 30::INTEGER,
     '{"database":{"metadataDatabaseId":1,"tableName":"orders","watermarkColumn":"id","watermarkType":"int","lastWatermark":"0","batchSize":100}}'::JSONB,
     now()::TIMESTAMPTZ, now()::TIMESTAMPTZ),

    -- Database polling trigger: watch for updated products
    ('Product Update Monitor'::TEXT,
     'Monitors the products table for updates based on updated_at timestamp'::TEXT,
     'database'::VARCHAR(20), 'paused'::VARCHAR(20), 1::BIGINT, 120::INTEGER,
     '{"database":{"metadataDatabaseId":1,"tableName":"products","watermarkColumn":"updated_at","watermarkType":"timestamp","lastWatermark":"2024-01-01T00:00:00Z","selectColumns":["id","name","price","updated_at"],"batchSize":50}}'::JSONB,
     now()::TIMESTAMPTZ, now()::TIMESTAMPTZ),

    -- Webhook trigger
    ('Incoming Webhook'::TEXT,
     'Receives webhook events from external systems'::TEXT,
     'webhook'::VARCHAR(20), 'active'::VARCHAR(20), 2::BIGINT, 0::INTEGER,
     '{"webhook":{"secret":"whsec_sample_secret_key_123","requiredHeaders":{"X-Webhook-Source":"external-system"}}}'::JSONB,
     now()::TIMESTAMPTZ, now()::TIMESTAMPTZ)
) AS seed(name, description, type, status, creator_id, polling_interval, config, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM trigger LIMIT 1);

-- ============================================================
-- Trigger Rules
-- ============================================================
INSERT INTO trigger_rule (trigger_id, name, conditions, created_at, updated_at)
SELECT * FROM (VALUES
    -- Rule for New Orders Watcher: only trigger if amount > 100
    (1::BIGINT, 'High Value Orders'::TEXT,
     '{"all":[{"field":"payload.amount","operator":"gt","value":100}]}'::JSONB,
     now()::TIMESTAMPTZ, now()::TIMESTAMPTZ),

    -- Rule for Product Update Monitor: only trigger if price changed
    (2::BIGINT, 'Price Changed'::TEXT,
     '{"any":[{"field":"payload.price","operator":"gt","value":0},{"field":"payload.status","operator":"eq","value":"active"}]}'::JSONB,
     now()::TIMESTAMPTZ, now()::TIMESTAMPTZ)
) AS seed(trigger_id, name, conditions, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM trigger_rule LIMIT 1);

-- ============================================================
-- Trigger Jobs (link triggers to jobs)
-- ============================================================
INSERT INTO trigger_job (trigger_id, job_id, priority, active, pass_event_data, created_at)
SELECT * FROM (VALUES
    -- New Orders Watcher triggers Customer ETL
    (1::BIGINT, 1::BIGINT, 0::INTEGER, true::BOOLEAN, true::BOOLEAN, now()::TIMESTAMPTZ),
    -- New Orders Watcher also triggers Daily Sales Report
    (1::BIGINT, 2::BIGINT, 1::INTEGER, true::BOOLEAN, false::BOOLEAN, now()::TIMESTAMPTZ),
    -- Product Update Monitor triggers Data Quality Check
    (2::BIGINT, 3::BIGINT, 0::INTEGER, true::BOOLEAN, true::BOOLEAN, now()::TIMESTAMPTZ),
    -- Webhook triggers Customer ETL
    (3::BIGINT, 1::BIGINT, 0::INTEGER, true::BOOLEAN, true::BOOLEAN, now()::TIMESTAMPTZ)
) AS seed(trigger_id, job_id, priority, active, pass_event_data, created_at)
WHERE NOT EXISTS (SELECT 1 FROM trigger_job LIMIT 1);

-- ============================================================
-- Trigger Executions (sample history)
-- ============================================================
INSERT INTO trigger_execution (trigger_id, started_at, finished_at, status, event_count, jobs_triggered, error, event_sample)
SELECT * FROM (VALUES
    -- Successful execution
    (1::BIGINT,
     (now() - interval '2 hours')::TIMESTAMPTZ,
     (now() - interval '2 hours' + interval '3 seconds')::TIMESTAMPTZ,
     'completed'::VARCHAR(20), 5::INTEGER, 2::INTEGER, ''::TEXT,
     '{"id":1001,"customer":"John Doe","amount":250.00}'::JSONB),

    -- Execution with no events
    (1::BIGINT,
     (now() - interval '1 hour')::TIMESTAMPTZ,
     (now() - interval '1 hour' + interval '1 second')::TIMESTAMPTZ,
     'no_events'::VARCHAR(20), 0::INTEGER, 0::INTEGER, ''::TEXT,
     NULL::JSONB),

    -- Failed execution
    (2::BIGINT,
     (now() - interval '30 minutes')::TIMESTAMPTZ,
     (now() - interval '30 minutes' + interval '5 seconds')::TIMESTAMPTZ,
     'failed'::VARCHAR(20), 0::INTEGER, 0::INTEGER,
     'connection refused: could not connect to database'::TEXT,
     NULL::JSONB),

    -- Recent successful execution
    (1::BIGINT,
     (now() - interval '5 minutes')::TIMESTAMPTZ,
     (now() - interval '5 minutes' + interval '2 seconds')::TIMESTAMPTZ,
     'completed'::VARCHAR(20), 3::INTEGER, 2::INTEGER, ''::TEXT,
     '{"id":1006,"customer":"Jane Smith","amount":175.50}'::JSONB)
) AS seed(trigger_id, started_at, finished_at, status, event_count, jobs_triggered, error, event_sample)
WHERE NOT EXISTS (SELECT 1 FROM trigger_execution LIMIT 1);

-- ============================================================
-- Summary
-- ============================================================
DO $$
BEGIN
    RAISE NOTICE 'Seed data loaded:';
    RAISE NOTICE '  - users: %',        (SELECT count(*) FROM users);
    RAISE NOTICE '  - jobs: %',          (SELECT count(*) FROM job);
    RAISE NOTICE '  - job_user_access: %', (SELECT count(*) FROM job_user_access);
    RAISE NOTICE '  - nodes: %',         (SELECT count(*) FROM node);
    RAISE NOTICE '  - ports: %',         (SELECT count(*) FROM port);
    RAISE NOTICE '  - triggers: %',      (SELECT count(*) FROM trigger);
    RAISE NOTICE '  - trigger_rules: %', (SELECT count(*) FROM trigger_rule);
    RAISE NOTICE '  - trigger_jobs: %',  (SELECT count(*) FROM trigger_job);
    RAISE NOTICE '  - trigger_executions: %', (SELECT count(*) FROM trigger_execution);
END $$;
