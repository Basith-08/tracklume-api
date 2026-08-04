package issue

import (
	"context"
	"fmt"
	"strings"

	"github.com/Basith-08/tracklume-api/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db database.Querier }

func NewRepository(db database.Querier) *Repository { return &Repository{db: db} }

const issueColumns = `id,project_id,sequence_number,identifier,title,description,type,status,priority,assignee_id,reporter_id,due_date,position,created_at,updated_at`

func scanIssue(row pgx.Row) (Issue, error) {
	var i Issue
	err := row.Scan(&i.ID, &i.ProjectID, &i.SequenceNumber, &i.Identifier, &i.Title, &i.Description, &i.Type, &i.Status, &i.Priority, &i.AssigneeID, &i.ReporterID, &i.DueDate, &i.Position, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}
func (r *Repository) Get(ctx context.Context, q database.Querier, projectID, issueID uuid.UUID) (Issue, error) {
	return scanIssue(q.QueryRow(ctx, `SELECT `+issueColumns+` FROM issues WHERE project_id=$1 AND id=$2 AND deleted_at IS NULL`, projectID, issueID))
}
func (r *Repository) IsMember(ctx context.Context, q database.Querier, projectID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id=$1 AND user_id=$2)`, projectID, userID).Scan(&exists)
	return exists, err
}
func (r *Repository) Create(ctx context.Context, tx pgx.Tx, input Issue) (Issue, error) {
	var seq int64
	err := tx.QueryRow(ctx, `UPDATE project_issue_counters SET next_sequence=next_sequence+1 WHERE project_id=$1 RETURNING next_sequence-1`, input.ProjectID).Scan(&seq)
	if err != nil {
		return Issue{}, err
	}
	input.SequenceNumber = seq
	input.Identifier = fmt.Sprintf("%s-%d", input.Identifier, seq)
	var position int64
	err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(position)+1,0) FROM issues WHERE project_id=$1 AND status=$2 AND deleted_at IS NULL`, input.ProjectID, input.Status).Scan(&position)
	if err != nil {
		return Issue{}, err
	}
	input.Position = position
	return scanIssue(tx.QueryRow(ctx, `INSERT INTO issues(id,project_id,sequence_number,identifier,title,description,type,status,priority,assignee_id,reporter_id,due_date,position) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING `+issueColumns, input.ID, input.ProjectID, input.SequenceNumber, input.Identifier, input.Title, input.Description, input.Type, input.Status, input.Priority, input.AssigneeID, input.ReporterID, input.DueDate, input.Position))
}
func (r *Repository) Update(ctx context.Context, q database.Querier, i Issue) (Issue, error) {
	return scanIssue(q.QueryRow(ctx, `UPDATE issues SET title=$3,description=$4,type=$5,status=$6,priority=$7,assignee_id=$8,due_date=$9,position=$10 WHERE project_id=$1 AND id=$2 AND deleted_at IS NULL RETURNING `+issueColumns, i.ProjectID, i.ID, i.Title, i.Description, i.Type, i.Status, i.Priority, i.AssigneeID, i.DueDate, i.Position))
}
func (r *Repository) SoftDelete(ctx context.Context, q database.Querier, projectID, issueID uuid.UUID) error {
	tag, err := q.Exec(ctx, `UPDATE issues SET deleted_at=now() WHERE project_id=$1 AND id=$2 AND deleted_at IS NULL`, projectID, issueID)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *Repository) Activity(ctx context.Context, q database.Querier, a Activity) error {
	_, err := q.Exec(ctx, `INSERT INTO issue_activities(id,issue_id,actor_id,action,field_name,old_value,new_value) VALUES($1,$2,$3,$4,$5,$6,$7)`, a.ID, a.IssueID, a.ActorID, a.Action, a.FieldName, a.OldValue, a.NewValue)
	return err
}
func (r *Repository) Activities(ctx context.Context, issueID uuid.UUID) ([]Activity, error) {
	rows, err := r.db.Query(ctx, `SELECT id,issue_id,actor_id,action,field_name,old_value,new_value,created_at FROM issue_activities WHERE issue_id=$1 ORDER BY created_at DESC`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Activity
	for rows.Next() {
		var a Activity
		if err = rows.Scan(&a.ID, &a.IssueID, &a.ActorID, &a.Action, &a.FieldName, &a.OldValue, &a.NewValue, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (r *Repository) MovePosition(ctx context.Context, tx pgx.Tx, current Issue, newStatus string, newPosition int64) (Issue, error) {
	if newStatus == current.Status {
		if newPosition > current.Position {
			_, _ = tx.Exec(ctx, `UPDATE issues SET position=position-1 WHERE project_id=$1 AND status=$2 AND position>$3 AND position<=$4 AND deleted_at IS NULL`, current.ProjectID, current.Status, current.Position, newPosition)
		} else if newPosition < current.Position {
			_, _ = tx.Exec(ctx, `UPDATE issues SET position=position+1 WHERE project_id=$1 AND status=$2 AND position>=$3 AND position<$4 AND deleted_at IS NULL`, current.ProjectID, current.Status, newPosition, current.Position)
		}
	} else {
		_, _ = tx.Exec(ctx, `UPDATE issues SET position=position-1 WHERE project_id=$1 AND status=$2 AND position>$3 AND deleted_at IS NULL`, current.ProjectID, current.Status, current.Position)
		_, _ = tx.Exec(ctx, `UPDATE issues SET position=position+1 WHERE project_id=$1 AND status=$2 AND position>=$3 AND deleted_at IS NULL`, current.ProjectID, newStatus, newPosition)
	}
	return scanIssue(tx.QueryRow(ctx, `UPDATE issues SET status=$3,position=$4 WHERE project_id=$1 AND id=$2 AND deleted_at IS NULL RETURNING `+issueColumns, current.ProjectID, current.ID, newStatus, newPosition))
}
func (r *Repository) List(ctx context.Context, projectID uuid.UUID, filter Filter) ([]Issue, int, error) {
	args := []any{projectID}
	where := []string{"i.project_id=$1", "i.deleted_at IS NULL"}
	add := func(sql string, v any) { args = append(args, v); where = append(where, fmt.Sprintf(sql, len(args))) }
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(i.title ILIKE $%d OR i.identifier ILIKE $%d)", n, n))
	}
	if len(filter.Statuses) > 0 {
		add(`i.status = ANY($%d)`, filter.Statuses)
	}
	if filter.Priority != "" {
		add(`i.priority=$%d`, filter.Priority)
	}
	if filter.Type != "" {
		add(`i.type=$%d`, filter.Type)
	}
	if filter.AssigneeID != nil {
		add(`i.assignee_id=$%d`, *filter.AssigneeID)
	} else if filter.Unassigned {
		where = append(where, "i.assignee_id IS NULL")
	}
	if filter.ReporterID != nil {
		add(`i.reporter_id=$%d`, *filter.ReporterID)
	}
	if !filter.DueBefore.IsZero() {
		add(`i.due_date <= $%d`, filter.DueBefore)
	}
	if !filter.DueAfter.IsZero() {
		add(`i.due_date >= $%d`, filter.DueAfter)
	}
	order := "i.created_at DESC"
	switch filter.Sort {
	case "updated_at":
		order = "i.updated_at DESC"
	case "priority":
		order = "CASE i.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,i.created_at DESC"
	case "due_date":
		order = "i.due_date ASC NULLS LAST,i.created_at DESC"
	}
	query := `SELECT ` + issueColumns + `,COUNT(*) OVER() FROM issues i WHERE ` + strings.Join(where, " AND ") + ` ORDER BY ` + order + ` LIMIT $` + fmt.Sprint(len(args)+1) + ` OFFSET $` + fmt.Sprint(len(args)+2)
	args = append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Issue
	total := 0
	for rows.Next() {
		var i Issue
		if err = rows.Scan(&i.ID, &i.ProjectID, &i.SequenceNumber, &i.Identifier, &i.Title, &i.Description, &i.Type, &i.Status, &i.Priority, &i.AssigneeID, &i.ReporterID, &i.DueDate, &i.Position, &i.CreatedAt, &i.UpdatedAt, &total); err != nil {
			return nil, 0, err
		}
		out = append(out, i)
	}
	return out, total, rows.Err()
}
