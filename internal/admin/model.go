package admin

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                 uuid.UUID
	Name               string
	Email              string
	AvatarURL          *string
	PlatformRole       string
	IsActive           bool
	LastLoginAt        *time.Time
	DeactivatedAt      *time.Time
	DeactivationReason *string
	DeletedAt          *time.Time
	CreatedAt          time.Time
	OwnedProjects      int64
	MemberProjects     int64
	ReportedIssues     int64
}

type Overview struct {
	TotalUsers    int64
	ActiveUsers   int64
	InactiveUsers int64
	DeletedUsers  int64
	NewUsers7d    int64
	ActiveUsers7d int64
	TotalProjects int64
	ActiveIssues  int64
}

type Filter struct {
	Search  string
	Status  string
	Page    int
	PerPage int
}
