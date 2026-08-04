package issue

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	app "github.com/Basith-08/tracklume-api/internal/apperror"
	"github.com/Basith-08/tracklume-api/internal/project"
	"github.com/Basith-08/tracklume-api/internal/validation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Filter struct {
	Search                 string
	Statuses               []string
	Priority, Type         string
	AssigneeID, ReporterID *uuid.UUID
	Unassigned             bool
	DueBefore, DueAfter    time.Time
	Sort                   string
	Page, PerPage          int
}
type Service struct {
	repo     *Repository
	projects *project.Service
	pool     *pgxpool.Pool
}

func NewService(repo *Repository, projects *project.Service, pool *pgxpool.Pool) *Service {
	return &Service{repo: repo, projects: projects, pool: pool}
}

func parseDate(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", *value)
	if err != nil {
		return nil, app.ErrValidation
	}
	return &parsed, nil
}
func parseUUID(value *string) (*uuid.UUID, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(*value)
	if err != nil {
		return nil, app.ErrValidation
	}
	return &parsed, nil
}
func activity(issueID, actorID uuid.UUID, action, field string, oldValue, newValue *string) Activity {
	return Activity{ID: uuid.New(), IssueID: issueID, ActorID: actorID, Action: action, FieldName: optionalString(field), OldValue: oldValue, NewValue: newValue}
}
func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func valueString(value any) *string {
	if value == nil {
		return nil
	}
	s := strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(toString(value)), "<nil>", ""))
	if s == "" {
		return nil
	}
	return &s
}
func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case *uuid.UUID:
		if v != nil {
			return v.String()
		}
	case *time.Time:
		if v != nil {
			return v.Format("2006-01-02")
		}
	case int64:
		return strconv.FormatInt(v, 10)
	}
	return ""
}

func (s *Service) Create(ctx context.Context, actorID, projectID uuid.UUID, req CreateRequest) (Issue, error) {
	if err := validation.Validator.Struct(req); err != nil {
		return Issue{}, app.ErrValidation
	}
	_, _, err := s.projects.Require(ctx, projectID, actorID, "issue_write")
	if err != nil {
		return Issue{}, err
	}
	assignee, err := parseUUID(req.AssigneeID)
	if err != nil {
		return Issue{}, err
	}
	due, err := parseDate(req.DueDate)
	if err != nil {
		return Issue{}, err
	}
	if assignee != nil {
		member, memberErr := s.repo.IsMember(ctx, s.pool, projectID, *assignee)
		if memberErr != nil {
			return Issue{}, memberErr
		}
		if !member {
			return Issue{}, app.ErrConflict
		}
	}
	p, _, err := s.projects.Access(ctx, projectID, actorID)
	if err != nil {
		return Issue{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Issue{}, err
	}
	defer tx.Rollback(ctx)
	created, err := s.repo.Create(ctx, tx, Issue{ID: uuid.New(), ProjectID: projectID, Identifier: p.Key, Title: strings.TrimSpace(req.Title), Description: req.Description, Type: req.Type, Status: req.Status, Priority: req.Priority, AssigneeID: assignee, ReporterID: actorID, DueDate: due})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Issue{}, app.ErrConflict
		}
		return Issue{}, err
	}
	if err = s.repo.Activity(ctx, tx, activity(created.ID, actorID, "created", "", nil, nil)); err != nil {
		return Issue{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Issue{}, err
	}
	return created, nil
}

func (s *Service) Get(ctx context.Context, actorID, projectID, issueID uuid.UUID) (Issue, error) {
	if _, _, err := s.projects.Require(ctx, projectID, actorID, "read"); err != nil {
		return Issue{}, err
	}
	i, err := s.repo.Get(ctx, s.pool, projectID, issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Issue{}, app.ErrNotFound
	}
	return i, err
}
func (s *Service) List(ctx context.Context, actorID, projectID uuid.UUID, filter Filter) ([]Issue, int, error) {
	if _, _, err := s.projects.Require(ctx, projectID, actorID, "read"); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, projectID, filter)
}

func (s *Service) Update(ctx context.Context, actorID, projectID, issueID uuid.UUID, input UpdateInput) (Issue, error) {
	_, _, err := s.projects.Require(ctx, projectID, actorID, "issue_write")
	if err != nil {
		return Issue{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Issue{}, err
	}
	defer tx.Rollback(ctx)
	current, err := s.repo.Get(ctx, tx, projectID, issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Issue{}, app.ErrNotFound
	}
	if err != nil {
		return Issue{}, err
	}
	next := current
	if input.Title != nil {
		next.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		next.Description = *input.Description
	}
	if input.Type != nil {
		next.Type = *input.Type
	}
	if input.Status != nil {
		next.Status = *input.Status
	}
	if input.Priority != nil {
		next.Priority = *input.Priority
	}
	if input.AssigneeSet {
		next.AssigneeID, err = parseUUID(input.AssigneeID)
		if err != nil {
			return Issue{}, err
		}
	}
	if input.DueDateSet {
		next.DueDate, err = parseDate(input.DueDate)
		if err != nil {
			return Issue{}, err
		}
	}
	if next.AssigneeID != nil {
		member, memberErr := s.repo.IsMember(ctx, tx, projectID, *next.AssigneeID)
		if memberErr != nil {
			return Issue{}, memberErr
		}
		if !member {
			return Issue{}, app.ErrConflict
		}
	}
	if next.Status != current.Status {
		moved, moveErr := s.repo.MovePosition(ctx, tx, current, next.Status, 0)
		if moveErr != nil {
			return Issue{}, moveErr
		}
		next.Position = moved.Position
	}
	updated, err := s.repo.Update(ctx, tx, next)
	if err != nil {
		return Issue{}, err
	}
	changes := []struct{ name, oldValue, newValue string }{{"title", current.Title, next.Title}, {"status", current.Status, next.Status}, {"priority", current.Priority, next.Priority}, {"assignee_id", toString(current.AssigneeID), toString(next.AssigneeID)}, {"due_date", toString(current.DueDate), toString(next.DueDate)}}
	for _, change := range changes {
		if change.oldValue != change.newValue {
			oldValue, newValue := valueString(change.oldValue), valueString(change.newValue)
			if err = s.repo.Activity(ctx, tx, activity(updated.ID, actorID, "updated", change.name, oldValue, newValue)); err != nil {
				return Issue{}, err
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Issue{}, err
	}
	return updated, nil
}

func (s *Service) Move(ctx context.Context, actorID, projectID, issueID uuid.UUID, status string, position int64) (Issue, error) {
	_, _, err := s.projects.Require(ctx, projectID, actorID, "issue_write")
	if err != nil {
		return Issue{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Issue{}, err
	}
	defer tx.Rollback(ctx)
	current, err := s.repo.Get(ctx, tx, projectID, issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Issue{}, app.ErrNotFound
	}
	if err != nil {
		return Issue{}, err
	}
	updated, err := s.repo.MovePosition(ctx, tx, current, status, position)
	if err != nil {
		return Issue{}, err
	}
	if current.Status != updated.Status {
		oldValue, newValue := current.Status, updated.Status
		if err = s.repo.Activity(ctx, tx, activity(updated.ID, actorID, "updated", "status", &oldValue, &newValue)); err != nil {
			return Issue{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Issue{}, err
	}
	return updated, nil
}
func (s *Service) Delete(ctx context.Context, actorID, projectID, issueID uuid.UUID) error {
	_, _, err := s.projects.Require(ctx, projectID, actorID, "issue_write")
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	current, err := s.repo.Get(ctx, tx, projectID, issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.ErrNotFound
	}
	if err != nil {
		return err
	}
	if err = s.repo.SoftDelete(ctx, tx, projectID, issueID); err != nil {
		return err
	}
	if err = s.repo.Activity(ctx, tx, activity(current.ID, actorID, "deleted", "", nil, nil)); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
func (s *Service) Activities(ctx context.Context, actorID, projectID, issueID uuid.UUID) ([]Activity, error) {
	if _, err := s.Get(ctx, actorID, projectID, issueID); err != nil {
		return nil, err
	}
	return s.repo.Activities(ctx, issueID)
}
