package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	app "github.com/Basith-08/tracklume-api/internal/apperror"
	"github.com/Basith-08/tracklume-api/internal/middleware"
	"github.com/Basith-08/tracklume-api/internal/response"
	"github.com/Basith-08/tracklume-api/internal/validation"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func decode(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("request contains multiple JSON values")
	}
	return nil
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decode(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	user, err := h.service.Register(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	response.Write(w, 201, presentUser(user))
}
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decode(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	login, err := h.service.Login(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	response.Write(w, 200, login)
}
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	id, _ := middleware.UserIDFromContext(r.Context())
	user, err := h.service.Profile(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	response.Write(w, 200, presentUser(user))
}
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	id, _ := middleware.UserIDFromContext(r.Context())
	var req UpdateProfileRequest
	if err := decode(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	user, err := h.service.UpdateProfile(r.Context(), id, req)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	response.Write(w, 200, presentUser(user))
}
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id, _ := middleware.UserIDFromContext(r.Context())
	var req ChangePasswordRequest
	if err := decode(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	if err := h.service.ChangePassword(r.Context(), id, req); err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	response.Write(w, 200, map[string]string{"message": "password updated"})
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr validationError
	switch {
	case errors.As(err, &validationErr):
		response.WriteError(w, r, 422, "VALIDATION_ERROR", "Request validation failed", validation.Fields(validationErr.err))
	case errors.Is(err, app.ErrConflict):
		response.WriteError(w, r, 409, "CONFLICT", "Resource already exists", nil)
	case errors.Is(err, app.ErrUnauthorized):
		response.WriteError(w, r, 401, "UNAUTHORIZED", "Invalid email or password", nil)
	case errors.Is(err, app.ErrInactive):
		response.WriteError(w, r, 403, "ACCOUNT_INACTIVE", "This account is inactive. Please contact support.", nil)
	case errors.Is(err, app.ErrNotFound):
		response.WriteError(w, r, 404, "NOT_FOUND", "Resource not found", nil)
	default:
		response.WriteError(w, r, 500, "INTERNAL_ERROR", "An unexpected error occurred", nil)
	}
}
