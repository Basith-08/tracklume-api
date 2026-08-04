package project

import (
	"github.com/google/uuid"
	"time"
)

type Project struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Key         string    `json:"key"`
	Description string    `json:"description"`
	OwnerID     uuid.UUID `json:"owner_id"`
	IsArchived  bool      `json:"is_archived"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type Member struct {
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL *string   `json:"avatar_url"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func Present(p Project) any {
	return map[string]any{"id": p.ID.String(), "name": p.Name, "key": p.Key, "description": p.Description, "owner_id": p.OwnerID.String(), "is_archived": p.IsArchived, "created_at": p.CreatedAt.UTC().Format(time.RFC3339Nano), "updated_at": p.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}
func PresentMember(m Member) any {
	return map[string]any{"user_id": m.UserID.String(), "name": m.Name, "email": m.Email, "avatar_url": m.AvatarURL, "role": m.Role, "created_at": m.CreatedAt.UTC().Format(time.RFC3339Nano)}
}
