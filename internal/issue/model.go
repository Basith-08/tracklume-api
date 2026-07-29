package issue

import (
	"github.com/google/uuid"
	"time"
)

type Issue struct {
	ID             uuid.UUID  `json:"id"`
	ProjectID      uuid.UUID  `json:"project_id"`
	SequenceNumber int64      `json:"sequence_number"`
	Identifier     string     `json:"identifier"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	Priority       string     `json:"priority"`
	AssigneeID     *uuid.UUID `json:"assignee_id"`
	ReporterID     uuid.UUID  `json:"reporter_id"`
	DueDate        *time.Time `json:"due_date"`
	Position       int64      `json:"position"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
type Activity struct {
	ID        uuid.UUID `json:"id"`
	IssueID   uuid.UUID `json:"issue_id"`
	ActorID   uuid.UUID `json:"actor_id"`
	Action    string    `json:"action"`
	FieldName *string   `json:"field_name"`
	OldValue  *string   `json:"old_value"`
	NewValue  *string   `json:"new_value"`
	CreatedAt time.Time `json:"created_at"`
}

func Present(i Issue) any {
	var due any
	if i.DueDate != nil {
		due = i.DueDate.Format("2006-01-02")
	}
	var assignee any
	if i.AssigneeID != nil {
		assignee = i.AssigneeID.String()
	}
	return map[string]any{"id": i.ID.String(), "project_id": i.ProjectID.String(), "sequence_number": i.SequenceNumber, "identifier": i.Identifier, "title": i.Title, "description": i.Description, "type": i.Type, "status": i.Status, "priority": i.Priority, "assignee_id": assignee, "reporter_id": i.ReporterID.String(), "due_date": due, "position": i.Position, "created_at": i.CreatedAt.UTC().Format(time.RFC3339Nano), "updated_at": i.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}
func PresentActivity(a Activity) any {
	return map[string]any{"id": a.ID.String(), "issue_id": a.IssueID.String(), "actor_id": a.ActorID.String(), "action": a.Action, "field_name": a.FieldName, "old_value": a.OldValue, "new_value": a.NewValue, "created_at": a.CreatedAt.UTC().Format(time.RFC3339Nano)}
}
