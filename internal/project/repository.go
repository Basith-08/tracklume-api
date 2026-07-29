package project

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

func scanProject(row pgx.Row) (Project, error) {
	var p Project
	err := row.Scan(&p.ID, &p.Name, &p.Key, &p.Description, &p.OwnerID, &p.IsArchived, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}
func (r *Repository) Create(ctx context.Context, p Project) (Project, error) {
	return scanProject(r.db.QueryRow(ctx, `INSERT INTO projects(id,name,key,description,owner_id) VALUES($1,$2,$3,$4,$5) RETURNING id,name,key,description,owner_id,is_archived,created_at,updated_at`, p.ID, p.Name, p.Key, p.Description, p.OwnerID))
}
func (r *Repository) Find(ctx context.Context, id, userID uuid.UUID) (Project, string, error) {
	var p Project
	var role string
	err := r.db.QueryRow(ctx, `SELECT p.id,p.name,p.key,p.description,p.owner_id,p.is_archived,p.created_at,p.updated_at,pm.role FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.user_id=$2 WHERE p.id=$1`, id, userID).Scan(&p.ID, &p.Name, &p.Key, &p.Description, &p.OwnerID, &p.IsArchived, &p.CreatedAt, &p.UpdatedAt, &role)
	return p, role, err
}
func (r *Repository) List(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]Project, error) {
	query := `SELECT p.id,p.name,p.key,p.description,p.owner_id,p.is_archived,p.created_at,p.updated_at FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.user_id=$1 WHERE ($2 OR p.is_archived=false) ORDER BY p.updated_at DESC`
	rows, err := r.db.Query(ctx, query, userID, includeArchived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Project
	for rows.Next() {
		var p Project
		if err = rows.Scan(&p.ID, &p.Name, &p.Key, &p.Description, &p.OwnerID, &p.IsArchived, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
func (r *Repository) Update(ctx context.Context, id uuid.UUID, name, description string) (Project, error) {
	return scanProject(r.db.QueryRow(ctx, `UPDATE projects SET name=$2,description=$3 WHERE id=$1 RETURNING id,name,key,description,owner_id,is_archived,created_at,updated_at`, id, name, description))
}
func (r *Repository) Archive(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `UPDATE projects SET is_archived=true WHERE id=$1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *Repository) Members(ctx context.Context, id uuid.UUID) ([]Member, error) {
	rows, err := r.db.Query(ctx, `SELECT u.id,u.name,u.email,u.avatar_url,pm.role,pm.created_at FROM project_members pm JOIN users u ON u.id=pm.user_id WHERE pm.project_id=$1 ORDER BY CASE pm.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 WHEN 'member' THEN 2 ELSE 3 END,u.name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Member
	for rows.Next() {
		var m Member
		if err = rows.Scan(&m.UserID, &m.Name, &m.Email, &m.AvatarURL, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}
func (r *Repository) FindUserByEmail(ctx context.Context, email string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE email=$1`, strings.ToLower(strings.TrimSpace(email))).Scan(&id)
	return id, err
}
func (r *Repository) AddMember(ctx context.Context, id, userID uuid.UUID, role string) (Member, error) {
	var m Member
	err := r.db.QueryRow(ctx, `INSERT INTO project_members(project_id,user_id,role) VALUES($1,$2,$3) RETURNING user_id,created_at`, id, userID, role).Scan(&m.UserID, &m.CreatedAt)
	if err != nil {
		return m, err
	}
	err = r.db.QueryRow(ctx, `SELECT u.name,u.email,u.avatar_url,pm.role FROM users u JOIN project_members pm ON pm.user_id=u.id WHERE pm.project_id=$1 AND pm.user_id=$2`, id, userID).Scan(&m.Name, &m.Email, &m.AvatarURL, &m.Role)
	return m, err
}
func (r *Repository) UpdateMember(ctx context.Context, id, userID uuid.UUID, role string) (Member, error) {
	var m Member
	err := r.db.QueryRow(ctx, `UPDATE project_members SET role=$3 WHERE project_id=$1 AND user_id=$2 RETURNING user_id,created_at`, id, userID, role).Scan(&m.UserID, &m.CreatedAt)
	if err != nil {
		return m, err
	}
	err = r.db.QueryRow(ctx, `SELECT u.name,u.email,u.avatar_url,pm.role FROM users u JOIN project_members pm ON pm.user_id=u.id WHERE pm.project_id=$1 AND pm.user_id=$2`, id, userID).Scan(&m.Name, &m.Email, &m.AvatarURL, &m.Role)
	return m, err
}
func (r *Repository) RemoveMember(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM project_members WHERE project_id=$1 AND user_id=$2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func IsNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
