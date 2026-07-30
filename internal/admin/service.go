package admin

import (
	"context"
	"errors"
	"strings"

	app "github.com/Basith-08/tracklume-api/internal/apperror"
	"github.com/Basith-08/tracklume-api/internal/validation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) authorize(ctx context.Context, actorID uuid.UUID) error {
	allowed, err := s.repo.IsSuperadmin(ctx, actorID)
	if err != nil {
		return err
	}
	if !allowed {
		return app.ErrForbidden
	}
	return nil
}

func (s *Service) Overview(ctx context.Context, actorID uuid.UUID) (Overview, error) {
	if err := s.authorize(ctx, actorID); err != nil {
		return Overview{}, err
	}
	return s.repo.Overview(ctx)
}

func (s *Service) ListUsers(ctx context.Context, actorID uuid.UUID, filter Filter) ([]User, int, error) {
	if err := s.authorize(ctx, actorID); err != nil {
		return nil, 0, err
	}
	if filter.Status != "" && filter.Status != "all" && filter.Status != "active" && filter.Status != "inactive" && filter.Status != "deleted" {
		return nil, 0, validationError{fields: map[string][]string{"status": {"Must be all, active, inactive, or deleted"}}}
	}
	return s.repo.List(ctx, filter)
}

func (s *Service) GetUser(ctx context.Context, actorID, targetID uuid.UUID) (User, error) {
	if err := s.authorize(ctx, actorID); err != nil {
		return User{}, err
	}
	user, err := s.repo.Get(ctx, targetID)
	if errorsIsNoRows(err) {
		return User{}, app.ErrNotFound
	}
	return user, err
}

func (s *Service) UpdateStatus(ctx context.Context, actorID, targetID uuid.UUID, req UpdateStatusRequest) (User, error) {
	if err := validation.Validator.Struct(req); err != nil {
		return User{}, validationError{fields: map[string][]string{"reason": {err.Error()}}}
	}
	if err := s.authorize(ctx, actorID); err != nil {
		return User{}, err
	}
	if actorID == targetID {
		return User{}, app.ErrForbidden
	}
	target, err := s.repo.Get(ctx, targetID)
	if errorsIsNoRows(err) {
		return User{}, app.ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	if target.PlatformRole == "superadmin" {
		return User{}, app.ErrForbidden
	}
	if target.DeletedAt != nil {
		return User{}, app.ErrConflict
	}
	var reason *string
	if !req.IsActive {
		value := strings.TrimSpace(req.Reason)
		if value == "" {
			return User{}, validationError{fields: map[string][]string{"reason": {"Reason is required when deactivating an account"}}}
		}
		reason = &value
	}
	user, err := s.repo.SetStatus(ctx, targetID, req.IsActive, reason)
	if errorsIsNoRows(err) {
		return User{}, app.ErrNotFound
	}
	return user, err
}

func (s *Service) DeleteUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	if err := s.authorize(ctx, actorID); err != nil {
		return err
	}
	if actorID == targetID {
		return app.ErrForbidden
	}
	target, err := s.repo.Get(ctx, targetID)
	if errorsIsNoRows(err) {
		return app.ErrNotFound
	}
	if err != nil {
		return err
	}
	if target.PlatformRole == "superadmin" {
		return app.ErrForbidden
	}
	if err = s.repo.SoftDelete(ctx, targetID); errorsIsNoRows(err) {
		return app.ErrNotFound
	}
	return err
}

func (s *Service) RestoreUser(ctx context.Context, actorID, targetID uuid.UUID) (User, error) {
	if err := s.authorize(ctx, actorID); err != nil {
		return User{}, err
	}
	if actorID == targetID {
		return User{}, app.ErrForbidden
	}
	target, err := s.repo.Get(ctx, targetID)
	if errorsIsNoRows(err) {
		return User{}, app.ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	if target.PlatformRole == "superadmin" {
		return User{}, app.ErrForbidden
	}
	user, err := s.repo.Restore(ctx, targetID)
	if errorsIsNoRows(err) {
		return User{}, app.ErrNotFound
	}
	return user, err
}

type validationError struct{ fields map[string][]string }

func (e validationError) Error() string               { return "request validation failed" }
func (e validationError) Unwrap() error               { return app.ErrValidation }
func (e validationError) Fields() map[string][]string { return e.fields }
func errorsIsNoRows(err error) bool                   { return errors.Is(err, pgx.ErrNoRows) }
