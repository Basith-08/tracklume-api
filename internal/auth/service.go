package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	app "github.com/Basith-08/tracklume-api/internal/apperror"
	"github.com/Basith-08/tracklume-api/internal/security"
	"github.com/Basith-08/tracklume-api/internal/validation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo   *Repository
	tokens *security.TokenManager
}

func NewService(repo *Repository, tokens *security.TokenManager) *Service {
	return &Service{repo: repo, tokens: tokens}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (User, error) {
	if err := validation.Validator.Struct(req); err != nil {
		return User{}, fmtValidation(err)
	}
	hash, err := security.HashPassword(req.Password)
	if err != nil {
		return User{}, err
	}
	user, err := s.repo.Create(ctx, User{ID: uuid.New(), Name: strings.TrimSpace(req.Name), Email: strings.ToLower(strings.TrimSpace(req.Email)), PasswordHash: hash})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, app.ErrConflict
		}
	}
	return user, err
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	if err := validation.Validator.Struct(req); err != nil {
		return LoginResponse{}, fmtValidation(err)
	}
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil || !security.VerifyPassword(user.PasswordHash, req.Password) {
		return LoginResponse{}, app.ErrUnauthorized
	}
	token, expires, err := s.tokens.Create(user.ID, time.Now().UTC())
	if err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{AccessToken: token, TokenType: "Bearer", ExpiresIn: expires, User: presentUser(user)}, nil
}

func (s *Service) Profile(ctx context.Context, id uuid.UUID) (User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if IsNoRows(err) {
		return User{}, app.ErrNotFound
	}
	return user, err
}

func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (User, error) {
	if err := validation.Validator.Struct(req); err != nil {
		return User{}, fmtValidation(err)
	}
	current, err := s.Profile(ctx, id)
	if err != nil {
		return User{}, err
	}
	name := current.Name
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	avatar := current.AvatarURL
	if req.AvatarURL != nil {
		avatar = req.AvatarURL
	}
	user, err := s.repo.UpdateProfile(ctx, id, name, avatar)
	if IsNoRows(err) {
		return User{}, app.ErrNotFound
	}
	return user, err
}

func (s *Service) ChangePassword(ctx context.Context, id uuid.UUID, req ChangePasswordRequest) error {
	if err := validation.Validator.Struct(req); err != nil {
		return fmtValidation(err)
	}
	user, err := s.Profile(ctx, id)
	if err != nil {
		return err
	}
	if !security.VerifyPassword(user.PasswordHash, req.CurrentPassword) {
		return app.ErrUnauthorized
	}
	hash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, id, hash)
}

type validationError struct{ err error }

func (e validationError) Error() string { return e.err.Error() }
func (e validationError) Unwrap() error { return app.ErrValidation }
func fmtValidation(err error) error     { return validationError{err: err} }
