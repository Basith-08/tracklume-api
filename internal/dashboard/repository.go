package dashboard

import (
	"context"

	"github.com/Basith-08/tracklume-api/internal/issue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Build(ctx context.Context, projectID uuid.UUID) (Summary, error) {
	summary := Summary{ByStatus: map[string]int{}, ByPriority: map[string]int{}, ByType: map[string]int{}}
	var done int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FILTER (WHERE status <> 'cancelled'), COUNT(*) FILTER (WHERE status='done'), COUNT(*) FILTER (WHERE status <> 'cancelled' AND due_date < CURRENT_DATE), COUNT(*) FILTER (WHERE status <> 'cancelled' AND due_date >= CURRENT_DATE AND due_date <= CURRENT_DATE+7) FROM issues WHERE project_id=$1 AND deleted_at IS NULL`, projectID).Scan(&summary.TotalActive, &done, &summary.Overdue, &summary.DueNext7Days)
	if err != nil {
		return summary, err
	}

	if err = r.group(ctx, projectID, "status", summary.ByStatus); err != nil {
		return summary, err
	}
	if err = r.group(ctx, projectID, "priority", summary.ByPriority); err != nil {
		return summary, err
	}
	if err = r.group(ctx, projectID, "type", summary.ByType); err != nil {
		return summary, err
	}
	summary.ProgressPercentage = CalculateProgress(done, summary.TotalActive)

	rows, err := r.pool.Query(ctx, `SELECT id,project_id,sequence_number,identifier,title,description,type,status,priority,assignee_id,reporter_id,due_date,position,created_at,updated_at FROM issues WHERE project_id=$1 AND deleted_at IS NULL ORDER BY updated_at DESC LIMIT 5`, projectID)
	if err != nil {
		return summary, err
	}
	defer rows.Close()
	var recent []issue.Issue
	for rows.Next() {
		var item issue.Issue
		if err = rows.Scan(&item.ID, &item.ProjectID, &item.SequenceNumber, &item.Identifier, &item.Title, &item.Description, &item.Type, &item.Status, &item.Priority, &item.AssigneeID, &item.ReporterID, &item.DueDate, &item.Position, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return summary, err
		}
		recent = append(recent, item)
	}
	summary.RecentlyUpdated = presentRecent(recent)
	return summary, rows.Err()
}

func (r *Repository) group(ctx context.Context, projectID uuid.UUID, field string, target map[string]int) error {
	// field is selected only from this internal allowlist.
	if field != "status" && field != "priority" && field != "type" {
		return nil
	}
	rows, err := r.pool.Query(ctx, "SELECT "+field+",COUNT(*) FROM issues WHERE project_id=$1 AND deleted_at IS NULL GROUP BY "+field, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var count int
		if err = rows.Scan(&key, &count); err != nil {
			return err
		}
		target[key] = count
	}
	return rows.Err()
}
