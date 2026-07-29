package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/Basith-08/tracklume-api/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db database.Querier }

func NewRepository(db database.Querier) *Repository { return &Repository{db: db} }

const userColumns = `id, name, email, password_hash, avatar_url, created_at, updated_at`

func scanUser(row pgx.Row) (User, error) {
	var user User
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (r *Repository) Create(ctx context.Context, user User) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `INSERT INTO users (id, name, email, password_hash, avatar_url) VALUES ($1,$2,$3,$4,$5) RETURNING `+userColumns, user.ID, user.Name, user.Email, user.PasswordHash, user.AvatarURL))
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email=$1`, strings.ToLower(strings.TrimSpace(email))))
}
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id=$1`, id))
}

func (r *Repository) UpdateProfile(ctx context.Context, id uuid.UUID, name string, avatarURL *string) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `UPDATE users SET name=$2, avatar_url=$3 WHERE id=$1 RETURNING `+userColumns, id, name, avatarURL))
}
func (r *Repository) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET password_hash=$2 WHERE id=$1`, id, hash)
	return err
}

func IsNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
