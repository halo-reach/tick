-- Tick Platform Database Schema
-- Generated from migrations 001-022
-- Run: psql -U postgres -d tick -f schema.sql


-- === 001_create_tenants.up.sql ===
CREATE TABLE tenants (
    id              VARCHAR(32) PRIMARY KEY,
    name            VARCHAR(255),
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    quota_max_tasks INT NOT NULL DEFAULT 100,
    quota_max_rps   INT NOT NULL DEFAULT 50,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- === 002_create_api_keys.up.sql ===
CREATE TABLE api_keys (
    id          VARCHAR(32) PRIMARY KEY,
    tenant_id   VARCHAR(32) NOT NULL REFERENCES tenants(id),
    key_hash    VARCHAR(64) NOT NULL,
    key_prefix  VARCHAR(8) NOT NULL,
    status      VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ
);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash) WHERE status = 'active';

-- === 003_create_targets.up.sql ===
CREATE TABLE targets (
    id          VARCHAR(32) PRIMARY KEY,
    tenant_id   VARCHAR(32) NOT NULL REFERENCES tenants(id),
    name        VARCHAR(255),
    type        VARCHAR(32) NOT NULL,
    config      JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_targets_tenant ON targets(tenant_id);

-- === 004_create_tasks.up.sql ===
CREATE TABLE tasks (
    id              VARCHAR(32) PRIMARY KEY,
    tenant_id       VARCHAR(32) NOT NULL REFERENCES tenants(id),
    name            VARCHAR(255) NOT NULL,
    schedule_type   VARCHAR(16) NOT NULL,
    cron_expr       VARCHAR(64),
    interval_value  INT,
    interval_unit   VARCHAR(4),
    once_at         TIMESTAMPTZ,
    target_id       VARCHAR(32) NOT NULL REFERENCES targets(id),
    timeout_secs    INT NOT NULL DEFAULT 30,
    retry_count     INT NOT NULL DEFAULT 3,
    missed_policy   VARCHAR(16) NOT NULL DEFAULT 'fire_once',
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    next_trigger_at TIMESTAMPTZ,
    total_executions BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX idx_tasks_tenant ON tasks(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_next_trigger ON tasks(next_trigger_at) WHERE status = 'active';

-- === 005_create_executions.up.sql ===
CREATE TABLE executions (
    id            BIGSERIAL,
    task_id       VARCHAR(32) NOT NULL,
    tenant_id     VARCHAR(32) NOT NULL,
    trigger_time  TIMESTAMPTZ NOT NULL,
    attempt       INT NOT NULL DEFAULT 1,
    status        VARCHAR(16) NOT NULL,
    status_code   INT,
    duration_ms   INT,
    request_body  TEXT,
    response_body TEXT,
    error_msg     TEXT,
    is_makeup     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create current and next month partitions
DO $$
DECLARE
    curr_start DATE := date_trunc('month', NOW());
    next_start DATE := date_trunc('month', NOW()) + INTERVAL '1 month';
    after_start DATE := date_trunc('month', NOW()) + INTERVAL '2 months';
BEGIN
    EXECUTE format(
        'CREATE TABLE executions_%s PARTITION OF executions FOR VALUES FROM (%L) TO (%L)',
        to_char(curr_start, 'YYYY_MM'), curr_start, next_start
    );
    EXECUTE format(
        'CREATE TABLE executions_%s PARTITION OF executions FOR VALUES FROM (%L) TO (%L)',
        to_char(next_start, 'YYYY_MM'), next_start, after_start
    );
END $$;

CREATE INDEX idx_executions_task ON executions(task_id, created_at DESC);
CREATE INDEX idx_executions_tenant ON executions(tenant_id, created_at DESC);

-- === 006_create_signing_secrets.up.sql ===
CREATE TABLE signing_secrets (
    id          VARCHAR(32) PRIMARY KEY,
    tenant_id   VARCHAR(32) NOT NULL REFERENCES tenants(id),
    secret      VARCHAR(64) NOT NULL,
    status      VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ
);
CREATE INDEX idx_signing_secrets_tenant ON signing_secrets(tenant_id) WHERE status = 'active';

-- === 007_create_audit_logs.up.sql ===
CREATE TABLE audit_logs (
    id            BIGSERIAL PRIMARY KEY,
    tenant_id     VARCHAR(32) NOT NULL,
    actor         VARCHAR(64) NOT NULL,
    action        VARCHAR(32) NOT NULL,
    resource_type VARCHAR(32) NOT NULL,
    resource_id   VARCHAR(32) NOT NULL,
    payload       JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_tenant ON audit_logs(tenant_id, created_at DESC);

-- === 008_add_tenant_auth.up.sql ===
ALTER TABLE tenants ADD COLUMN username VARCHAR(64) UNIQUE;
ALTER TABLE tenants ADD COLUMN password_hash VARCHAR(128);
ALTER TABLE tenants ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT false;

-- === 009_add_apikey_name.up.sql ===
ALTER TABLE api_keys ADD COLUMN name VARCHAR(255) NOT NULL DEFAULT 'default';

-- === 010_migrate_v1_tenants.up.sql ===
-- For existing tenants without username: generate username from tenant name
-- Password is a bcrypt hash of a random temp value; must_change_password forces reset on first login
UPDATE tenants
SET username = LOWER(REGEXP_REPLACE(name, '[^a-zA-Z0-9_-]', '_', 'g')) || '_' || SUBSTRING(id FROM 5 FOR 8),
    password_hash = '$2a$12$placeholder.needs.manual.reset.xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
    must_change_password = true
WHERE username IS NULL;

-- === 011_add_execution_is_manual.up.sql ===
ALTER TABLE executions ADD COLUMN is_manual BOOLEAN NOT NULL DEFAULT false;

-- === 012_add_execution_protection.up.sql ===
ALTER TABLE tasks
    ADD COLUMN concurrency_policy VARCHAR(10) NOT NULL DEFAULT 'skip',
    ADD COLUMN max_concurrency INT NOT NULL DEFAULT 1,
    ADD COLUMN retry_backoff VARCHAR(20) NOT NULL DEFAULT 'exponential',
    ADD COLUMN execution_retention_days INT NOT NULL DEFAULT 30;

-- === 013_create_credentials.up.sql ===
CREATE TABLE IF NOT EXISTS credentials (
    id VARCHAR(32) PRIMARY KEY,
    tenant_id VARCHAR(32) NOT NULL REFERENCES tenants(id),
    name VARCHAR(128) NOT NULL,
    type VARCHAR(32) NOT NULL,
    config BYTEA NOT NULL,
    config_preview JSONB NOT NULL DEFAULT '{}',
    timeout_secs INT NOT NULL DEFAULT 10,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_credentials_tenant_status ON credentials (tenant_id, status) WHERE status != 'deleted';

-- === 014_create_target_credentials.up.sql ===
CREATE TABLE IF NOT EXISTS target_credentials (
    id VARCHAR(32) PRIMARY KEY,
    target_id VARCHAR(32) NOT NULL REFERENCES targets(id),
    credential_id VARCHAR(32) NOT NULL REFERENCES credentials(id),
    inject_location VARCHAR(16) NOT NULL,
    inject_key VARCHAR(128) NOT NULL,
    inject_prefix VARCHAR(64) NOT NULL DEFAULT '',
    order_index INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_target_credentials_target ON target_credentials (target_id);
CREATE UNIQUE INDEX uk_target_cred_order ON target_credentials (target_id, order_index);

-- === 015_add_task_hooks.up.sql ===
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS pre_hooks JSONB NOT NULL DEFAULT '[]';
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS post_hooks JSONB NOT NULL DEFAULT '[]';

-- === 016_add_execution_hooks.up.sql ===
ALTER TABLE executions ADD COLUMN IF NOT EXISTS triggered_by VARCHAR(16) NOT NULL DEFAULT 'scheduler';
ALTER TABLE executions ADD COLUMN IF NOT EXISTS hooks_result JSONB;

-- === 017_add_credential_code.up.sql ===
ALTER TABLE credentials ADD COLUMN code VARCHAR(64) NOT NULL DEFAULT '';

CREATE UNIQUE INDEX idx_credentials_tenant_code ON credentials (tenant_id, code) WHERE code != '' AND status != 'deleted';

-- === 018_add_request_headers.up.sql ===
ALTER TABLE executions ADD COLUMN IF NOT EXISTS request_headers TEXT DEFAULT '';

-- === 019_drop_target_credentials.up.sql ===
DROP TABLE IF EXISTS target_credentials;
-- === 020_create_variables.up.sql ===
CREATE TABLE IF NOT EXISTS variables (
    id VARCHAR(32) PRIMARY KEY,
    tenant_id VARCHAR(32) NOT NULL REFERENCES tenants(id),
    key VARCHAR(128) NOT NULL,
    value TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, key)
);
CREATE INDEX idx_variables_tenant ON variables(tenant_id);

-- === 021_create_users_and_members.up.sql ===
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(32) PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(128) NOT NULL,
    display_name VARCHAR(255),
    email VARCHAR(255) UNIQUE,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    failed_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_members (
    id VARCHAR(32) PRIMARY KEY,
    tenant_id VARCHAR(32) NOT NULL REFERENCES tenants(id),
    user_id VARCHAR(32) NOT NULL REFERENCES users(id),
    role VARCHAR(16) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, user_id)
);

CREATE TABLE IF NOT EXISTS invitations (
    id VARCHAR(32) PRIMARY KEY,
    tenant_id VARCHAR(32) NOT NULL REFERENCES tenants(id),
    code VARCHAR(8) UNIQUE NOT NULL,
    created_by VARCHAR(32) NOT NULL REFERENCES users(id),
    role VARCHAR(16) NOT NULL DEFAULT 'member',
    max_uses INT NOT NULL DEFAULT 0,
    used_count INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_tenant_members_user ON tenant_members(user_id);
CREATE INDEX idx_tenant_members_tenant ON tenant_members(tenant_id);
CREATE INDEX idx_invitations_tenant ON invitations(tenant_id);
CREATE INDEX idx_invitations_code ON invitations(code);

-- === 022_migrate_tenant_users.up.sql ===
INSERT INTO users (id, username, password_hash, display_name, status, created_at)
SELECT
    'usr_' || encode(gen_random_bytes(12), 'hex'),
    username,
    password_hash,
    name,
    'active',
    created_at
FROM tenants
WHERE username IS NOT NULL AND username != '';

INSERT INTO tenant_members (id, tenant_id, user_id, role, joined_at)
SELECT
    'mbr_' || encode(gen_random_bytes(12), 'hex'),
    t.id,
    u.id,
    'owner',
    t.created_at
FROM tenants t
JOIN users u ON u.username = t.username
WHERE t.username IS NOT NULL AND t.username != '';

-- End of schema
