package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	app "github.com/Basith-08/tracklume-api/internal/apperror"
	"github.com/Basith-08/tracklume-api/internal/middleware"
	"github.com/Basith-08/tracklume-api/internal/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.service.Overview(r.Context(), actorID(r))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.Write(w, http.StatusOK, overview)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, perPage := response.NormalizePagination(parseInt(query.Get("page")), parseInt(query.Get("per_page")))
	users, total, err := h.service.ListUsers(r.Context(), actorID(r), Filter{Search: query.Get("search"), Status: query.Get("status"), Page: page, PerPage: perPage})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	items := make([]UserResponse, 0, len(users))
	for _, user := range users {
		items = append(items, PresentUser(user))
	}
	response.WriteCollection(w, http.StatusOK, items, response.PaginationMeta(page, perPage, total))
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "User ID is invalid", nil)
		return
	}
	user, err := h.service.GetUser(r.Context(), actorID(r), targetID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.Write(w, http.StatusOK, PresentUser(user))
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "User ID is invalid", nil)
		return
	}
	var req UpdateStatusRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	user, err := h.service.UpdateStatus(r.Context(), actorID(r), targetID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.Write(w, http.StatusOK, PresentUser(user))
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "User ID is invalid", nil)
		return
	}
	if err = h.service.DeleteUser(r.Context(), actorID(r), targetID); err != nil {
		h.writeError(w, r, err)
		return
	}
	response.WriteNoContent(w)
}

func (h *Handler) RestoreUser(w http.ResponseWriter, r *http.Request) {
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "User ID is invalid", nil)
		return
	}
	user, err := h.service.RestoreUser(r.Context(), actorID(r), targetID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.Write(w, http.StatusOK, PresentUser(user))
}

func actorID(r *http.Request) uuid.UUID {
	id, _ := middleware.UserIDFromContext(r.Context())
	return id
}

func parseInt(value string) int {
	parsed, _ := strconv.Atoi(value)
	return parsed
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr validationError
	if errors.As(err, &validationErr) {
		response.WriteError(w, r, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Request validation failed", validationErr.Fields())
		return
	}
	switch {
	case errors.Is(err, app.ErrForbidden):
		response.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Platform administrator access is required", nil)
	case errors.Is(err, app.ErrNotFound):
		response.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "User not found", nil)
	case errors.Is(err, app.ErrConflict):
		response.WriteError(w, r, http.StatusConflict, "CONFLICT", "The requested account state conflicts with existing data", nil)
	default:
		response.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", nil)
	}
}
