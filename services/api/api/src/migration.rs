/// Database migrations — all CREATE TABLE IF NOT EXISTS statements.
pub async fn migrate(db: &sqlx::PgPool) {
    sqlx::query(
        "CREATE TABLE IF NOT EXISTS workspace (
            id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
            name       TEXT        NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT now()
        )",
    )
    .execute(db)
    .await
    .expect("failed to create workspace table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS source (
            id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
            workspace_id UUID        NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
            name         TEXT        NOT NULL,
            source_type  TEXT        NOT NULL,
            config       JSONB       NOT NULL DEFAULT '{}',
            created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
        )",
    )
    .execute(db)
    .await
    .expect("failed to create source table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS users (
            id            BIGSERIAL   PRIMARY KEY,
            email         TEXT        NOT NULL UNIQUE,
            password      TEXT        NOT NULL,
            prenom        TEXT        NOT NULL DEFAULT '',
            nom           TEXT        NOT NULL DEFAULT '',
            role          TEXT        NOT NULL DEFAULT 'user',
            actif         BOOLEAN     NOT NULL DEFAULT true,
            refresh_token TEXT,
            created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
            updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
            deleted_at    TIMESTAMPTZ
        )",
    )
    .execute(db)
    .await
    .expect("failed to create users table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS metadata_database (
            id            BIGSERIAL   PRIMARY KEY,
            name          TEXT        NOT NULL,
            host          TEXT        NOT NULL,
            port          INTEGER     NOT NULL,
            \"user\"      TEXT        NOT NULL,
            password      TEXT        NOT NULL,
            database_name TEXT        NOT NULL,
            ssl_mode      TEXT        NOT NULL DEFAULT 'disable',
            extra         TEXT        NOT NULL DEFAULT '',
            db_type       TEXT        NOT NULL
        )",
    )
    .execute(db)
    .await
    .expect("failed to create metadata_database table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS metadata_sftp (
            id          BIGSERIAL   PRIMARY KEY,
            name        TEXT        NOT NULL,
            host        TEXT        NOT NULL,
            port        INTEGER     NOT NULL,
            \"user\"    TEXT        NOT NULL,
            password    TEXT        NOT NULL,
            private_key TEXT        NOT NULL DEFAULT '',
            base_path   TEXT        NOT NULL DEFAULT '',
            extra       TEXT        NOT NULL DEFAULT ''
        )",
    )
    .execute(db)
    .await
    .expect("failed to create metadata_sftp table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS metadata_email (
            id        BIGSERIAL   PRIMARY KEY,
            name      TEXT        NOT NULL,
            imap_host TEXT        NOT NULL,
            imap_port INTEGER     NOT NULL,
            smtp_host TEXT        NOT NULL,
            smtp_port INTEGER     NOT NULL,
            username  TEXT        NOT NULL,
            password  TEXT        NOT NULL,
            use_tls   BOOLEAN     NOT NULL DEFAULT true
        )",
    )
    .execute(db)
    .await
    .expect("failed to create metadata_email table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS job (
            id          BIGSERIAL   PRIMARY KEY,
            name        TEXT        NOT NULL,
            description TEXT        NOT NULL DEFAULT '',
            file_path   TEXT        NOT NULL DEFAULT '',
            creator_id  BIGINT      NOT NULL REFERENCES users(id),
            active      BOOLEAN     NOT NULL DEFAULT true,
            visibility  TEXT        NOT NULL DEFAULT 'private',
            output_path TEXT        NOT NULL DEFAULT '',
            created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
            updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
        )",
    )
    .execute(db)
    .await
    .expect("failed to create job table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS node (
            id      BIGSERIAL   PRIMARY KEY,
            job_id  BIGINT      NOT NULL REFERENCES job(id) ON DELETE CASCADE,
            type    TEXT        NOT NULL,
            name    TEXT        NOT NULL,
            xpos    FLOAT4      NOT NULL DEFAULT 0,
            ypos    FLOAT4      NOT NULL DEFAULT 0,
            data    JSONB       NOT NULL DEFAULT '{}'
        )",
    )
    .execute(db)
    .await
    .expect("failed to create node table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS port (
            id                BIGSERIAL   PRIMARY KEY,
            node_id           BIGINT      NOT NULL REFERENCES node(id) ON DELETE CASCADE,
            type              TEXT        NOT NULL,
            connected_node_id BIGINT      NOT NULL
        )",
    )
    .execute(db)
    .await
    .expect("failed to create port table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS job_user_access (
            job_id  BIGINT  NOT NULL REFERENCES job(id) ON DELETE CASCADE,
            user_id BIGINT  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            role    TEXT    NOT NULL DEFAULT 'viewer',
            PRIMARY KEY (job_id, user_id)
        )",
    )
    .execute(db)
    .await
    .expect("failed to create job_user_access table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS job_notification_contact (
            job_id  BIGINT  NOT NULL REFERENCES job(id) ON DELETE CASCADE,
            user_id BIGINT  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            PRIMARY KEY (job_id, user_id)
        )",
    )
    .execute(db)
    .await
    .expect("failed to create job_notification_contact table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS trigger (
            id               BIGSERIAL   PRIMARY KEY,
            name             TEXT        NOT NULL,
            description      TEXT        NOT NULL DEFAULT '',
            type             TEXT        NOT NULL,
            status           TEXT        NOT NULL DEFAULT 'inactive',
            creator_id       BIGINT      NOT NULL REFERENCES users(id),
            polling_interval INTEGER     NOT NULL DEFAULT 60,
            last_polled_at   TIMESTAMPTZ,
            last_error       TEXT        NOT NULL DEFAULT '',
            config           JSONB       NOT NULL DEFAULT '{}',
            created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
            updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
            deleted_at       TIMESTAMPTZ
        )",
    )
    .execute(db)
    .await
    .expect("failed to create trigger table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS trigger_rule (
            id         BIGSERIAL   PRIMARY KEY,
            trigger_id BIGINT      NOT NULL REFERENCES trigger(id) ON DELETE CASCADE,
            name       TEXT        NOT NULL,
            conditions JSONB       NOT NULL DEFAULT '{}',
            created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
            deleted_at TIMESTAMPTZ
        )",
    )
    .execute(db)
    .await
    .expect("failed to create trigger_rule table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS trigger_job (
            id              BIGSERIAL   PRIMARY KEY,
            trigger_id      BIGINT      NOT NULL REFERENCES trigger(id) ON DELETE CASCADE,
            job_id          BIGINT      NOT NULL REFERENCES job(id) ON DELETE CASCADE,
            priority        INTEGER     NOT NULL DEFAULT 0,
            active          BOOLEAN     NOT NULL DEFAULT true,
            pass_event_data BOOLEAN     NOT NULL DEFAULT false,
            created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
            deleted_at      TIMESTAMPTZ
        )",
    )
    .execute(db)
    .await
    .expect("failed to create trigger_job table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS trigger_execution (
            id             BIGSERIAL   PRIMARY KEY,
            trigger_id     BIGINT      NOT NULL REFERENCES trigger(id) ON DELETE CASCADE,
            started_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
            finished_at    TIMESTAMPTZ,
            status         TEXT        NOT NULL DEFAULT 'running',
            event_count    INTEGER     NOT NULL DEFAULT 0,
            jobs_triggered INTEGER     NOT NULL DEFAULT 0,
            error          TEXT        NOT NULL DEFAULT '',
            event_sample   JSONB
        )",
    )
    .execute(db)
    .await
    .expect("failed to create trigger_execution table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS dataset (
            id                   BIGSERIAL   PRIMARY KEY,
            name                 TEXT        NOT NULL,
            description          TEXT        NOT NULL DEFAULT '',
            creator_id           BIGINT      NOT NULL REFERENCES users(id),
            metadata_database_id BIGINT      NOT NULL REFERENCES metadata_database(id),
            query                TEXT        NOT NULL DEFAULT '',
            schema               JSONB       NOT NULL DEFAULT '{}',
            status               TEXT        NOT NULL DEFAULT 'pending',
            last_refreshed_at    TIMESTAMPTZ,
            last_error           TEXT        NOT NULL DEFAULT '',
            created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
            updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
            deleted_at           TIMESTAMPTZ
        )",
    )
    .execute(db)
    .await
    .expect("failed to create dataset table");

    tracing::info!("migrations applied");
}
