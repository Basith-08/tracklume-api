package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Basith-08/tracklume-api/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db database.Querier }

func NewRepository(db database.Querier) *Repository { return &Repository{db: db} }

const userSelect = `
SELECT u.id, u.name, u.email, u.avatar_url, u.platform_role, u.is_active,
       u.last_login_at, u.deactivated_at, u.deactivation_reason, u.deleted_at, u.created_at,
       (SELECT COUNT(*) FROM projects p WHERE p.owner_id=u.id),
       (SELECT COUNT(*) FROM project_members pm WHERE pm.user_id=u.id),
       (SELECT COUNT(*) FROM issues i WHERE i.reporter_id=u.id AND i.deleted_at IS NULL)
FROM users u`

func scanUser(row pgx.Row) (User, error) {
	var user User
	err := row.Scan(
		&user.ID, &user.Name, &user.Email, &user.AvatarURL, &user.PlatformRole,
		&user.IsActive, &user.LastLoginAt, &user.DeactivatedAt, &user.DeactivationReason, &user.DeletedAt,
		&user.CreatedAt, &user.OwnedProjects, &user.MemberProjects, &user.ReportedIssues,
	)
	return user, err
}

func (r *Repository) IsSuperadmin(ctx context.Context, id uuid.UUID) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT platform_role='superadmin' AND is_active AND deleted_at IS NULL FROM users WHERE id=$1`, id).Scan(&allowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return allowed, err
}

func (r *Repository) Overview(ctx context.Context) (Overview, error) {
	var overview Overview
	err := r.db.QueryRow(ctx, `
SELECT
    COUNT(*)::bigint,
    COUNT(*) FILTER (WHERE is_active)::bigint,
    COUNT(*) FILTER (WHERE NOT is_active AND deleted_at IS NULL)::bigint,
    COUNT(*) FILTER (WHERE deleted_at IS NOT NULL)::bigint,
    COUNT(*) FILTER (WHERE created_at >= now() - INTERVAL '7 days')::bigint,
    COUNT(*) FILTER (WHERE last_login_at >= now() - INTERVAL '7 days')::bigint,
    (SELECT COUNT(*) FROM projects)::bigint,
    (SELECT COUNT(*) FROM issues WHERE deleted_at IS NULL)::bigint
FROM users`).Scan(
		&overview.TotalUsers, &overview.ActiveUsers, &overview.InactiveUsers, &overview.DeletedUsers,
		&overview.NewUsers7d, &overview.ActiveUsers7d, &overview.TotalProjects,
		&overview.ActiveIssues,
	)
	return overview, err
}

func (r *Repository) List(ctx context.Context, filter Filter) ([]User, int, error) {
	where, args := buildWhere(filter)
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users u WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := userSelect + ` WHERE ` + where + ` ORDER BY u.last_login_at DESC NULLS LAST, u.created_at DESC LIMIT $` + fmt.Sprint(len(args)+1) + ` OFFSET $` + fmt.Sprint(len(args)+2)
	args = append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users := make([]User, 0, filter.PerPage)
	for rows.Next() {
		user, scanErr := scanUser(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		users = append(users, user)
	}
	return users, total, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (User, error) {
	return scanUser(r.db.QueryRow(ctx, userSelect+` WHERE u.id=$1`, id))
}

func (r *Repository) SetStatus(ctx context.Context, id uuid.UUID, active bool, reason *string) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `
UPDATE users
SET is_active=$2,
    deactivated_at=CASE WHEN $2 THEN NULL ELSE now() END,
    deactivation_reason=CASE WHEN $2 THEN NULL ELSE $3 END
WHERE id=$1 AND deleted_at IS NULL
RETURNING id, name, email, avatar_url, platform_role, is_active,
          last_login_at, deactivated_at, deactivation_reason, deleted_at, created_at,
          (SELECT COUNT(*) FROM projects p WHERE p.owner_id=users.id),
          (SELECT COUNT(*) FROM project_members pm WHERE pm.user_id=users.id),
          (SELECT COUNT(*) FROM issues i WHERE i.reporter_id=users.id AND i.deleted_at IS NULL)`, id, active, reason))
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET is_active=false, deleted_at=now(), deactivated_at=now(), deactivation_reason='Deleted by platform administrator' WHERE id=$1 AND deleted_at IS NULL`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

func (r *Repository) Restore(ctx context.Context, id uuid.UUID) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `
UPDATE users
SET is_active=true, deleted_at=NULL, deactivated_at=NULL, deactivation_reason=NULL
WHERE id=$1 AND deleted_at IS NOT NULL
RETURNING id, name, email, avatar_url, platform_role, is_active,
          last_login_at, deactivated_at, deactivation_reason, deleted_at, created_at,
          (SELECT COUNT(*) FROM projects p WHERE p.owner_id=users.id),
          (SELECT COUNT(*) FROM project_members pm WHERE pm.user_id=users.id),
          (SELECT COUNT(*) FROM issues i WHERE i.reporter_id=users.id AND i.deleted_at IS NULL)`, id))
}

func (r *Repository) BootstrapSuperadmin(ctx context.Context, name, email, passwordHash string) (User, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
INSERT INTO users (name, email, password_hash, platform_role, is_active)
VALUES ($1, lower(trim($2)), $3, 'superadmin', true)
ON CONFLICT (email) DO UPDATE SET
    platform_role='superadmin',
    is_active=true,
    deactivated_at=NULL,
    deactivation_reason=NULL,
    deleted_at=NULL
RETURNING id`, name, email, passwordHash).Scan(&id)
	if err != nil {
		return User{}, err
	}
	return r.Get(ctx, id)
}

func buildWhere(filter Filter) (string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 2)
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		where = append(where, fmt.Sprintf("(u.name ILIKE $%d OR u.email ILIKE $%d)", len(args), len(args)))
	}
	if filter.Status == "active" {
		where = append(where, "u.is_active=true AND u.deleted_at IS NULL")
	}
	if filter.Status == "inactive" {
		where = append(where, "u.is_active=false AND u.deleted_at IS NULL")
	}
	if filter.Status == "deleted" {
		where = append(where, "u.deleted_at IS NOT NULL")
	}
	return strings.Join(where, " AND "), args
}
