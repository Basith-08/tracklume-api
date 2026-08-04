package project

import (
	"encoding/json"
	"errors"
	"net/http"

	app "github.com/Basith-08/tracklume-api/internal/apperror"
	"github.com/Basith-08/tracklume-api/internal/middleware"
	"github.com/Basith-08/tracklume-api/internal/response"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func decodeJSON(r *http.Request, target any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	return nil
}
func idParam(r *http.Request) (uuid.UUID, error) { return uuid.Parse(chi.URLParam(r, "projectID")) }
func userID(r *http.Request) uuid.UUID           { id, _ := middleware.UserIDFromContext(r.Context()); return id }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), userID(r), r.URL.Query().Get("include_archived") == "true")
	if err != nil {
		h.error(w, r, err)
		return
	}
	out := make([]any, 0, len(items))
	for _, p := range items {
		out = append(out, Present(p))
	}
	response.WriteCollection(w, 200, out, response.PaginationMeta(1, len(out), len(out)))
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := decodeJSON(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	p, err := h.service.Create(r.Context(), userID(r), req)
	if err != nil {
		h.error(w, r, err)
		return
	}
	response.Write(w, 201, Present(p))
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Project ID is invalid", nil)
		return
	}
	p, _, err := h.service.Access(r.Context(), id, userID(r))
	if err != nil {
		h.error(w, r, err)
		return
	}
	response.Write(w, 200, Present(p))
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Project ID is invalid", nil)
		return
	}
	var req UpdateRequest
	if err = decodeJSON(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	p, err := h.service.Update(r.Context(), id, userID(r), req)
	if err != nil {
		h.error(w, r, err)
		return
	}
	response.Write(w, 200, Present(p))
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Project ID is invalid", nil)
		return
	}
	if err = h.service.Archive(r.Context(), id, userID(r)); err != nil {
		h.error(w, r, err)
		return
	}
	response.WriteNoContent(w)
}
func (h *Handler) Members(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Project ID is invalid", nil)
		return
	}
	items, err := h.service.Members(r.Context(), id, userID(r))
	if err != nil {
		h.error(w, r, err)
		return
	}
	out := make([]any, 0, len(items))
	for _, m := range items {
		out = append(out, PresentMember(m))
	}
	response.WriteCollection(w, 200, out, response.PaginationMeta(1, len(out), len(out)))
}
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Project ID is invalid", nil)
		return
	}
	var req AddMemberRequest
	if err = decodeJSON(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	m, err := h.service.AddMember(r.Context(), id, userID(r), req)
	if err != nil {
		h.error(w, r, err)
		return
	}
	response.Write(w, 201, PresentMember(m))
}
func (h *Handler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Project ID is invalid", nil)
		return
	}
	target, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "User ID is invalid", nil)
		return
	}
	var req UpdateMemberRequest
	if err = decodeJSON(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	m, err := h.service.UpdateMember(r.Context(), id, userID(r), target, req)
	if err != nil {
		h.error(w, r, err)
		return
	}
	response.Write(w, 200, PresentMember(m))
}
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Project ID is invalid", nil)
		return
	}
	target, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "User ID is invalid", nil)
		return
	}
	if err = h.service.RemoveMember(r.Context(), id, userID(r), target); err != nil {
		h.error(w, r, err)
		return
	}
	response.WriteNoContent(w)
}

func (h *Handler) error(w http.ResponseWriter, r *http.Request, err error) {
	fields := map[string][]string(nil)
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields = map[string][]string{"request": {ve.Error()}}
	}
	switch {
	case errors.Is(err, app.ErrValidation):
		response.WriteError(w, r, 422, "VALIDATION_ERROR", "Request validation failed", fields)
	case errors.Is(err, app.ErrForbidden):
		response.WriteError(w, r, 403, "FORBIDDEN", "You do not have permission for this project", nil)
	case errors.Is(err, app.ErrNotFound):
		response.WriteError(w, r, 404, "NOT_FOUND", "Resource not found", nil)
	case errors.Is(err, app.ErrConflict):
		response.WriteError(w, r, 409, "CONFLICT", "The requested state conflicts with existing data", nil)
	default:
		response.WriteError(w, r, 500, "INTERNAL_ERROR", "An unexpected error occurred", nil)
	}
}
