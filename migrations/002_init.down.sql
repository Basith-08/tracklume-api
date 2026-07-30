DROP INDEX IF EXISTS users_platform_role_idx;
DROP INDEX IF EXISTS users_active_last_login_idx;

ALTER TABLE users
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deactivation_reason,
    DROP COLUMN IF EXISTS deactivated_at,
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS platform_role;
