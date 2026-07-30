package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Basith-08/tracklume-api/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db database.Querier }

func NewRepository(db database.Querier) *Repository { return &Repository{db: db} }

const userColumns = `id, name, email, password_hash, avatar_url, platform_role, is_active, last_login_at, deactivated_at, deleted_at, created_at, updated_at`

func scanUser(row pgx.Row) (User, error) {
	var user User
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.AvatarURL, &user.PlatformRole, &user.IsActive, &user.LastLoginAt, &user.DeactivatedAt, &user.DeletedAt, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (r *Repository) Create(ctx context.Context, user User) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `INSERT INTO users (id, name, email, password_hash, avatar_url) VALUES ($1,$2,$3,$4,$5) RETURNING `+userColumns, user.ID, user.Name, user.Email, user.PasswordHash, user.AvatarURL))
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email=$1 AND deleted_at IS NULL`, strings.ToLower(strings.TrimSpace(email))))
}
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id=$1 AND deleted_at IS NULL`, id))
}

func (r *Repository) UpdateProfile(ctx context.Context, id uuid.UUID, name string, avatarURL *string) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `UPDATE users SET name=$2, avatar_url=$3 WHERE id=$1 RETURNING `+userColumns, id, name, avatarURL))
}
func (r *Repository) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET password_hash=$2 WHERE id=$1`, id, hash)
	return err
}

func (r *Repository) IsActive(ctx context.Context, id uuid.UUID) (bool, error) {
	var active bool
	err := r.db.QueryRow(ctx, `SELECT is_active AND deleted_at IS NULL FROM users WHERE id=$1`, id).Scan(&active)
	if IsNoRows(err) {
		return false, nil
	}
	return active, err
}

func (r *Repository) IsSuperadmin(ctx context.Context, id uuid.UUID) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT platform_role='superadmin' AND is_active AND deleted_at IS NULL FROM users WHERE id=$1`, id).Scan(&allowed)
	if IsNoRows(err) {
		return false, nil
	}
	return allowed, err
}

func (r *Repository) MarkLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET last_login_at=$2 WHERE id=$1 AND is_active=true AND deleted_at IS NULL`, id, at)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

func IsNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
