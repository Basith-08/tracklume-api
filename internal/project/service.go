package project

import (
	"context"
	"errors"
	"strings"

	app "github.com/Basith-08/tracklume-api/internal/apperror"
	"github.com/Basith-08/tracklume-api/internal/validation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func RoleAllows(role, action string) bool {
	switch action {
	case "read":
		return role == "owner" || role == "admin" || role == "member" || role == "viewer"
	case "issue_write":
		return role == "owner" || role == "admin" || role == "member"
	case "member_manage":
		return role == "owner" || role == "admin"
	case "project_write":
		return role == "owner" || role == "admin"
	case "archive":
		return role == "owner"
	default:
		return false
	}
}

func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, req CreateRequest) (Project, error) {
	if err := validation.Validator.Struct(req); err != nil {
		return Project{}, app.ErrValidation
	}
	key := strings.ToUpper(strings.TrimSpace(req.Key))
	if len(key) < 2 || len(key) > 10 {
		return Project{}, app.ErrValidation
	}
	p, err := s.repo.Create(ctx, Project{ID: uuid.New(), Name: strings.TrimSpace(req.Name), Key: key, Description: strings.TrimSpace(req.Description), OwnerID: ownerID})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Project{}, app.ErrConflict
		}
	}
	return p, err
}
func (s *Service) List(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]Project, error) {
	return s.repo.List(ctx, userID, includeArchived)
}
func (s *Service) Access(ctx context.Context, id, userID uuid.UUID) (Project, string, error) {
	p, role, err := s.repo.Find(ctx, id, userID)
	if IsNoRows(err) {
		return Project{}, "", app.ErrNotFound
	}
	return p, role, err
}
func (s *Service) Require(ctx context.Context, id, userID uuid.UUID, action string) (Project, string, error) {
	p, role, err := s.Access(ctx, id, userID)
	if err != nil {
		return p, role, err
	}
	if !RoleAllows(role, action) {
		return Project{}, "", app.ErrForbidden
	}
	if p.IsArchived && action != "read" {
		return Project{}, "", app.ErrConflict
	}
	return p, role, nil
}
func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, req UpdateRequest) (Project, error) {
	if err := validation.Validator.Struct(req); err != nil {
		return Project{}, app.ErrValidation
	}
	p, _, err := s.Require(ctx, id, userID, "project_write")
	if err != nil {
		return Project{}, err
	}
	name := p.Name
	desc := p.Description
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		desc = *req.Description
	}
	return s.repo.Update(ctx, id, name, strings.TrimSpace(desc))
}
func (s *Service) Archive(ctx context.Context, id, userID uuid.UUID) error {
	_, _, err := s.Require(ctx, id, userID, "archive")
	if err != nil {
		return err
	}
	err = s.repo.Archive(ctx, id)
	if IsNoRows(err) {
		return app.ErrNotFound
	}
	return err
}
func (s *Service) Members(ctx context.Context, id, userID uuid.UUID) ([]Member, error) {
	_, _, err := s.Require(ctx, id, userID, "read")
	if err != nil {
		return nil, err
	}
	return s.repo.Members(ctx, id)
}
func (s *Service) AddMember(ctx context.Context, id, userID uuid.UUID, req AddMemberRequest) (Member, error) {
	if err := validation.Validator.Struct(req); err != nil {
		return Member{}, app.ErrValidation
	}
	_, _, err := s.Require(ctx, id, userID, "member_manage")
	if err != nil {
		return Member{}, err
	}
	memberID, err := s.repo.FindUserByEmail(ctx, req.Email)
	if IsNoRows(err) {
		return Member{}, app.ErrNotFound
	}
	if err != nil {
		return Member{}, err
	}
	member, err := s.repo.AddMember(ctx, id, memberID, req.Role)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Member{}, app.ErrConflict
		}
	}
	return member, err
}
func (s *Service) UpdateMember(ctx context.Context, id, userID, targetID uuid.UUID, req UpdateMemberRequest) (Member, error) {
	if err := validation.Validator.Struct(req); err != nil {
		return Member{}, app.ErrValidation
	}
	p, actorRole, err := s.Require(ctx, id, userID, "member_manage")
	if err != nil {
		return Member{}, err
	}
	if targetID == p.OwnerID {
		return Member{}, app.ErrForbidden
	}
	if actorRole == "admin" && req.Role == "admin" {
		return Member{}, app.ErrForbidden
	}
	member, err := s.repo.UpdateMember(ctx, id, targetID, req.Role)
	if IsNoRows(err) {
		return Member{}, app.ErrNotFound
	}
	return member, err
}
func (s *Service) RemoveMember(ctx context.Context, id, userID, targetID uuid.UUID) error {
	p, role, err := s.Require(ctx, id, userID, "member_manage")
	if err != nil {
		return err
	}
	if targetID == p.OwnerID || (role == "admin" && targetID == userID) {
		return app.ErrForbidden
	}
	err = s.repo.RemoveMember(ctx, id, targetID)
	if IsNoRows(err) {
		return app.ErrNotFound
	}
	return err
}
