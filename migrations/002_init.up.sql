ALTER TABLE users
    ADD COLUMN platform_role TEXT NOT NULL DEFAULT 'user' CHECK (platform_role IN ('user', 'superadmin')),
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN last_login_at TIMESTAMPTZ,
    ADD COLUMN deactivated_at TIMESTAMPTZ,
    ADD COLUMN deactivation_reason TEXT,
    ADD COLUMN deleted_at TIMESTAMPTZ;

CREATE INDEX users_active_last_login_idx ON users(is_active, last_login_at DESC NULLS LAST);
CREATE INDEX users_platform_role_idx ON users(platform_role);
