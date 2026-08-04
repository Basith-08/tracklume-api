package admin

import "time"

type UpdateStatusRequest struct {
	IsActive bool   `json:"is_active"`
	Reason   string `json:"reason" validate:"max=500"`
}

type UserResponse struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Email              string  `json:"email"`
	AvatarURL          *string `json:"avatar_url"`
	PlatformRole       string  `json:"platform_role"`
	IsActive           bool    `json:"is_active"`
	LastLoginAt        *string `json:"last_login_at,omitempty"`
	DeactivatedAt      *string `json:"deactivated_at,omitempty"`
	DeactivationReason *string `json:"deactivation_reason,omitempty"`
	DeletedAt          *string `json:"deleted_at,omitempty"`
	CreatedAt          string  `json:"created_at"`
	OwnedProjects      int64   `json:"owned_projects"`
	MemberProjects     int64   `json:"member_projects"`
	ReportedIssues     int64   `json:"reported_issues"`
}

func PresentUser(user User) UserResponse {
	return UserResponse{
		ID:                 user.ID.String(),
		Name:               user.Name,
		Email:              user.Email,
		AvatarURL:          user.AvatarURL,
		PlatformRole:       user.PlatformRole,
		IsActive:           user.IsActive,
		LastLoginAt:        formatTime(user.LastLoginAt),
		DeactivatedAt:      formatTime(user.DeactivatedAt),
		DeactivationReason: user.DeactivationReason,
		DeletedAt:          formatTime(user.DeletedAt),
		CreatedAt:          user.CreatedAt.UTC().Format(time.RFC3339Nano),
		OwnedProjects:      user.OwnedProjects,
		MemberProjects:     user.MemberProjects,
		ReportedIssues:     user.ReportedIssues,
	}
}

func formatTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}
