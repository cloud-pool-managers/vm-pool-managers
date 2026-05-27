-- vm-registrar inventory table
-- Phase 3: auto-registration of student VMs
-- Applied on the same PostgreSQL instance as control_center

CREATE TABLE IF NOT EXISTS vm_instances (
    id              TEXT        PRIMARY KEY,           -- Nova UUID
    name            TEXT        NOT NULL,              -- Nova display name
    ip              TEXT        NOT NULL,              -- Primary private IPv4
    public_ip       TEXT        NOT NULL DEFAULT '',   -- Floating IP
    az              TEXT        NOT NULL DEFAULT '',   -- Availability zone
    role            TEXT        NOT NULL DEFAULT '',   -- web | db | worker
    app_port        INTEGER     NOT NULL DEFAULT 0,    -- Declared listening port
    environment     TEXT        NOT NULL DEFAULT '',   -- demo | prod | staging
    status          TEXT        NOT NULL DEFAULT 'starting',
    -- status: 'starting' | 'ready' | 'draining' | 'dead'
    healthy         BOOLEAN     NOT NULL DEFAULT false,
    activity_status TEXT        NOT NULL DEFAULT 'idle',
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    raw_meta        JSONB       NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_vm_role        ON vm_instances(role);
CREATE INDEX IF NOT EXISTS idx_vm_status      ON vm_instances(status);
CREATE INDEX IF NOT EXISTS idx_vm_healthy     ON vm_instances(healthy);
CREATE INDEX IF NOT EXISTS idx_vm_last_seen   ON vm_instances(last_seen DESC);

-- Guacamole connection identifier (populated by control center sync loop)
ALTER TABLE vm_instances ADD COLUMN IF NOT EXISTS guac_connection_id TEXT NOT NULL DEFAULT '';

-- Stale VMs view (heartbeat > 60s ago)
CREATE OR REPLACE VIEW vm_stale AS
    SELECT id, name, ip, role, status, last_seen,
           EXTRACT(EPOCH FROM (NOW() - last_seen))::INT AS stale_seconds
    FROM vm_instances
    WHERE last_seen < NOW() - INTERVAL '60 seconds'
    ORDER BY last_seen ASC;
