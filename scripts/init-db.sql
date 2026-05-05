-- ============================================
-- NL SCADA Center v1.0 – Database Schema & Seed
-- ============================================

-- Bật extension nếu cần (thường có sẵn)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Xóa bảng cũ (chỉ trong dev)
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;

-- ============================================
-- 1. TENANTS
-- ============================================
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 2. USERS
-- ============================================
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20) NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin','operator','viewer')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 3. DEVICES
-- ============================================
CREATE TABLE devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    device_type     VARCHAR(20) NOT NULL DEFAULT 'gateway' CHECK (device_type IN ('gateway','virtual')),
    mqtt_client_id  VARCHAR(255) UNIQUE,
    status          VARCHAR(20) NOT NULL DEFAULT 'offline' CHECK (status IN ('online','offline')),
    last_heartbeat  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 4. TAGS
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
-- 5. INDEXES
-- ============================================
CREATE INDEX IF NOT EXISTS idx_devices_tenant ON devices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tags_device ON tags(device_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);

-- ============================================
-- 6. SEED DATA
-- ============================================
-- Sử dụng DO block để tránh lỗi duplicate key
DO $$
BEGIN
    -- Tenant mẫu
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = '00000000-0000-0000-0000-000000000001') THEN
        INSERT INTO tenants (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'Demo Tenant');
    END IF;

    -- User admin (password: admin123)
    IF NOT EXISTS (SELECT 1 FROM users WHERE id = '00000000-0000-0000-0000-000000000010') THEN
        INSERT INTO users (id, tenant_id, email, password_hash, role)
        VALUES ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001',
                'admin@demo.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin');
    END IF;

    -- Device mẫu
    IF NOT EXISTS (SELECT 1 FROM devices WHERE id = '00000000-0000-0000-0000-000000000100') THEN
        INSERT INTO devices (id, tenant_id, name, device_type, mqtt_client_id, status)
        VALUES ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000001',
                'Virtual Device 1', 'virtual', 'gateway-sim', 'offline');
    END IF;

    -- Tags mẫu
    IF NOT EXISTS (SELECT 1 FROM tags WHERE id = '00000000-0000-0000-0000-000000001000') THEN
        INSERT INTO tags (id, device_id, name, data_type, unit, description) VALUES
        ('00000000-0000-0000-0000-000000001000', '00000000-0000-0000-0000-000000000100', 'temperature', 'float', '°C', 'Nhiệt độ phòng'),
        ('00000000-0000-0000-0000-000000001001', '00000000-0000-0000-0000-000000000100', 'humidity', 'float', '%', 'Độ ẩm'),
        ('00000000-0000-0000-0000-000000001002', '00000000-0000-0000-0000-000000000100', 'pump_status', 'bool', NULL, 'Trạng thái bơm');
    END IF;
END $$;