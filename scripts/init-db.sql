-- ============================================
-- NL SCADA Center – Database Schema (v1.1+)
-- ============================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Xóa bảng cũ (chỉ dùng trong môi trường dev)
DROP TABLE IF EXISTS site_retention_policies CASCADE;
DROP TABLE IF EXISTS system_event_logs CASCADE;
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS pid_widgets CASCADE;
DROP TABLE IF EXISTS pid_diagrams CASCADE;
DROP TABLE IF EXISTS alert_logs CASCADE;
DROP TABLE IF EXISTS control_logs CASCADE;
DROP TABLE IF EXISTS control_config CASCADE;
DROP TABLE IF EXISTS alert_rules CASCADE;
DROP TABLE IF EXISTS tags CASCADE;
DROP TABLE IF EXISTS devices CASCADE;
DROP TABLE IF EXISTS memberships CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS sites CASCADE;

-- ============================================
-- 1. SITES
-- ============================================
CREATE TABLE sites (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 2. USERS
-- ============================================
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    name          VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 3. MEMBERSHIPS
-- ============================================
CREATE TABLE memberships (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    site_id    UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin','operator','viewer','auditor')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, site_id)
);

-- ============================================
-- 4. DEVICES
-- ============================================
CREATE TABLE devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id         UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    device_type     VARCHAR(20) NOT NULL DEFAULT 'gateway' CHECK (device_type IN ('gateway','virtual')),
    mqtt_client_id  VARCHAR(255) UNIQUE,
    status          VARCHAR(20) NOT NULL DEFAULT 'offline' CHECK (status IN ('online','offline')),
    last_heartbeat  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 5. TAGS
-- ============================================
CREATE TABLE tags (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    data_type   VARCHAR(20) NOT NULL CHECK (data_type IN ('float','int','bool','string')),
    unit        VARCHAR(50),
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, name)
);

-- ============================================
-- 6. ALERT RULES
-- ============================================
CREATE TABLE alert_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id         UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    tag_id          UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    min_value       DOUBLE PRECISION,
    max_value       DOUBLE PRECISION,
    severity        VARCHAR(20) NOT NULL DEFAULT 'WARNING' CHECK (severity IN ('INFO','WARNING','CRITICAL')),
    message_template TEXT,
    is_enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ
);

-- ============================================
-- 7. CONTROL CONFIG
-- ============================================
CREATE TABLE control_config (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id         UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    tag_id          UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    control_type    VARCHAR(50) NOT NULL,
    allowed_values  JSONB,
    is_enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ
);

-- ============================================
-- 8. CONTROL LOGS
-- ============================================
CREATE TABLE control_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id          UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    device_id        UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tag_name         VARCHAR(100) NOT NULL,
    requested_value  VARCHAR(50) NOT NULL,
    previous_value   VARCHAR(50),
    status           VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','SENT','SUCCESS','FAILED','TIMEOUT')),
    error_message    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at          TIMESTAMPTZ,
    acknowledged_at  TIMESTAMPTZ
);

-- ============================================
-- 9. ALERT LOGS
-- ============================================
CREATE TABLE alert_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id          UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    device_id        UUID REFERENCES devices(id) ON DELETE CASCADE,
    tag_name         VARCHAR(100) NOT NULL,
    triggered_value  DOUBLE PRECISION NOT NULL,
    threshold_value  DOUBLE PRECISION NOT NULL,
    severity         VARCHAR(20) NOT NULL CHECK (severity IN ('INFO','WARNING','CRITICAL')),
    message          TEXT NOT NULL,
    status           VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','ACKNOWLEDGED','RESOLVED')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at  TIMESTAMPTZ,
    acknowledged_by  UUID REFERENCES users(id),
    resolved_at      TIMESTAMPTZ
);

-- ============================================
-- 10. P&ID DIAGRAMS
-- ============================================
CREATE TABLE pid_diagrams (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id       UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL,
    background_url TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ
);

-- ============================================
-- 11. P&ID WIDGETS
-- ============================================
CREATE TABLE pid_widgets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    diagram_id  UUID NOT NULL REFERENCES pid_diagrams(id) ON DELETE CASCADE,
    device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    tag_name    VARCHAR(100) NOT NULL,
    position_x  NUMERIC(5,2) NOT NULL,
    position_y  NUMERIC(5,2) NOT NULL,
    widget_type VARCHAR(20) NOT NULL CHECK (widget_type IN ('TEXT','PUMP','VALVE'))
);

-- ============================================
-- 12. AUDIT LOGS
-- ============================================
CREATE TABLE audit_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id          UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action_type      VARCHAR(50) NOT NULL,
    resource_target  VARCHAR(50) NOT NULL,
    target_id        UUID,
    description      TEXT NOT NULL,
    old_values       JSONB,
    new_values       JSONB,
    ip_address       VARCHAR(45) NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 13. SYSTEM EVENT LOGS
-- ============================================
CREATE TABLE system_event_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id         UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    device_id       UUID REFERENCES devices(id) ON DELETE CASCADE,
    event_type      VARCHAR(50) NOT NULL,
    severity_level  VARCHAR(20) NOT NULL CHECK (severity_level IN ('INFO','WARNING','ERROR')),
    message         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 14. SITE RETENTION POLICIES
-- ============================================
CREATE TABLE site_retention_policies (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id                UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE UNIQUE,
    audit_logs_days        INTEGER NOT NULL DEFAULT 90,
    system_event_logs_days INTEGER NOT NULL DEFAULT 90,
    alert_logs_days        INTEGER NOT NULL DEFAULT 180,
    telemetry_influx_days  INTEGER NOT NULL DEFAULT 30,
    updated_at             TIMESTAMPTZ,
    updated_by             UUID REFERENCES users(id)
);

-- ============================================
-- INDEXES
-- ============================================
CREATE INDEX idx_memberships_user ON memberships(user_id);
CREATE INDEX idx_memberships_site ON memberships(site_id);
CREATE INDEX idx_devices_site ON devices(site_id);
CREATE INDEX idx_devices_status ON devices(status);
CREATE INDEX idx_tags_device ON tags(device_id);
CREATE INDEX idx_alert_rules_tag ON alert_rules(tag_id);
CREATE INDEX idx_alert_rules_site ON alert_rules(site_id);
CREATE INDEX idx_control_config_tag ON control_config(tag_id);
CREATE INDEX idx_control_logs_site_device ON control_logs(site_id, device_id);
CREATE INDEX idx_alert_logs_site_device ON alert_logs(site_id, device_id);
CREATE INDEX idx_alert_logs_status ON alert_logs(status);
CREATE INDEX idx_pid_widgets_diagram ON pid_widgets(diagram_id);
CREATE INDEX idx_audit_logs_site ON audit_logs(site_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_system_event_logs_site ON system_event_logs(site_id);