-- ============================================
-- NL SCADA Center v1.0 – Database Schema & Seed
-- (site + membership model)
-- ============================================

-- Bật extension nếu cần (thường có sẵn)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Xóa bảng cũ (chỉ dùng trong môi trường dev)
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS sites;
DROP TABLE IF EXISTS tenants CASCADE; -- xóa bảng cũ nếu còn

-- ============================================
-- 1. SITES
-- ============================================
CREATE TABLE sites (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 2. USERS (toàn cục, không gắn trực tiếp site)
-- ============================================
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 3. MEMBERSHIPS (liên kết user - site - role)
-- ============================================
CREATE TABLE memberships (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    site_id    UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin','operator','viewer')),
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
-- 6. INDEXES
-- ============================================
CREATE INDEX IF NOT EXISTS idx_memberships_user ON memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_memberships_site ON memberships(site_id);
CREATE INDEX IF NOT EXISTS idx_devices_site ON devices(site_id);
CREATE INDEX IF NOT EXISTS idx_tags_device ON tags(device_id);

-- ============================================
-- 7. SEED DATA (dữ liệu mẫu)
-- ============================================
-- Sử dụng DO block để tránh lỗi duplicate key
DO $$
BEGIN
    -- Site mẫu
    IF NOT EXISTS (SELECT 1 FROM sites WHERE id = '00000000-0000-0000-0000-000000000001') THEN
        INSERT INTO sites (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'Demo Site');
    END IF;

    -- User admin (password: admin123, bcrypt hash)
    IF NOT EXISTS (SELECT 1 FROM users WHERE id = '00000000-0000-0000-0000-000000000010') THEN
        INSERT INTO users (id, email, password_hash) VALUES ('00000000-0000-0000-0000-000000000010', 'admin@demo.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy');
    END IF;

    -- Membership: admin trong site demo
    IF NOT EXISTS (SELECT 1 FROM memberships WHERE user_id = '00000000-0000-0000-0000-000000000010' AND site_id = '00000000-0000-0000-0000-000000000001') THEN
        INSERT INTO memberships (user_id, site_id, role) VALUES ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'admin');
    END IF;

    -- Device mẫu (thuộc site demo)
    IF NOT EXISTS (SELECT 1 FROM devices WHERE id = '00000000-0000-0000-0000-000000000100') THEN
        INSERT INTO devices (id, site_id, name, device_type, mqtt_client_id, status)
        VALUES ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000001', 'Virtual Device 1', 'virtual', 'gateway-sim', 'offline');
    END IF;

    -- Tags cho device mẫu
    IF NOT EXISTS (SELECT 1 FROM tags WHERE id = '00000000-0000-0000-0000-000000001000') THEN
        INSERT INTO tags (id, device_id, name, data_type, unit, description) VALUES
        ('00000000-0000-0000-0000-000000001000', '00000000-0000-0000-0000-000000000100', 'temperature', 'float', '°C', 'Nhiệt độ phòng'),
        ('00000000-0000-0000-0000-000000001001', '00000000-0000-0000-0000-000000000100', 'humidity', 'float', '%', 'Độ ẩm'),
        ('00000000-0000-0000-0000-000000001002', '00000000-0000-0000-0000-000000000100', 'pump_status', 'bool', NULL, 'Trạng thái bơm');
    END IF;
END $$;