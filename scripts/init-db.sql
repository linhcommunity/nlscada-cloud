-- ============================================
-- NL SCADA Center v1.0 – Database Schema & Seed
-- ============================================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS sites;

-- 1. Sites (thay cho tenants)
CREATE TABLE sites (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Users (không có tenant_id)
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email          VARCHAR(255) NOT NULL UNIQUE,
    password_hash  VARCHAR(255) NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Memberships (liên kết user - site - role)
CREATE TABLE memberships (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    site_id    UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin','operator','viewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, site_id)
);

-- 4. Devices (thay tenant_id bằng site_id)
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

-- 5. Tags (giữ nguyên)
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

-- Indexes
CREATE INDEX idx_devices_site ON devices(site_id);
CREATE INDEX idx_tags_device ON tags(device_id);
CREATE INDEX idx_memberships_user ON memberships(user_id);
CREATE INDEX idx_memberships_site ON memberships(site_id);

-- Seed data: user admin toàn cục (chưa thuộc site nào)
INSERT INTO users (id, email, password_hash, email_verified) VALUES 
    ('00000000-0000-0000-0000-000000000010', 'admin@demo.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', TRUE)
ON CONFLICT (id) DO NOTHING;