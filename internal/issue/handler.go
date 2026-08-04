package issue

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	app "github.com/Basith-08/tracklume-api/internal/apperror"
	"github.com/Basith-08/tracklume-api/internal/middleware"
	"github.com/Basith-08/tracklume-api/internal/response"
	"github.com/Basith-08/tracklume-api/internal/validation"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func decode(r *http.Request, target any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(target)
}
func params(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	issueID, err := uuid.Parse(chi.URLParam(r, "issueID"))
	return projectID, issueID, err
}
func actor(r *http.Request) uuid.UUID { id, _ := middleware.UserIDFromContext(r.Context()); return id }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Project ID is invalid", nil)
		return
	}
	var req CreateRequest
	if err = decode(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	i, err := h.service.Create(r.Context(), actor(r), projectID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.Write(w, 201, Present(i))
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	projectID, issueID, err := params(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Issue ID is invalid", nil)
		return
	}
	i, err := h.service.Get(r.Context(), actor(r), projectID, issueID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.Write(w, 200, Present(i))
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Project ID is invalid", nil)
		return
	}
	q := r.URL.Query()
	page, perPage := response.NormalizePagination(parseInt(q.Get("page")), parseInt(q.Get("per_page")))
	filter := Filter{Search: strings.TrimSpace(q.Get("search")), Priority: q.Get("priority"), Type: q.Get("type"), Sort: q.Get("sort"), Page: page, PerPage: perPage}
	filter.Statuses = csv(q.Get("status"))
	if value := q.Get("assignee_id"); value != "" && value != "unassigned" {
		id, parseErr := uuid.Parse(value)
		if parseErr != nil {
			response.WriteError(w, r, 422, "VALIDATION_ERROR", "Request validation failed", map[string][]string{"assignee_id": {"Must be a valid UUID"}})
			return
		}
		filter.AssigneeID = &id
	} else if q.Get("assignee_id") == "unassigned" {
		filter.Unassigned = true
	}
	if value := q.Get("reporter_id"); value != "" {
		id, parseErr := uuid.Parse(value)
		if parseErr != nil {
			response.WriteError(w, r, 422, "VALIDATION_ERROR", "Request validation failed", map[string][]string{"reporter_id": {"Must be a valid UUID"}})
			return
		}
		filter.ReporterID = &id
	}
	var parseErr error
	if value := q.Get("due_before"); value != "" {
		filter.DueBefore, parseErr = time.Parse("2006-01-02", value)
		if parseErr != nil {
			response.WriteError(w, r, 422, "VALIDATION_ERROR", "Request validation failed", nil)
			return
		}
	}
	if value := q.Get("due_after"); value != "" {
		filter.DueAfter, parseErr = time.Parse("2006-01-02", value)
		if parseErr != nil {
			response.WriteError(w, r, 422, "VALIDATION_ERROR", "Request validation failed", nil)
			return
		}
	}
	items, total, err := h.service.List(r.Context(), actor(r), projectID, filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]any, 0, len(items))
	for _, i := range items {
		out = append(out, Present(i))
	}
	response.WriteCollection(w, 200, out, response.PaginationMeta(page, perPage, total))
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	projectID, issueID, err := params(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Issue ID is invalid", nil)
		return
	}
	var req UpdateRequest
	if err = decode(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	input := UpdateInput{Title: req.Title, Description: req.Description, Type: req.Type, Status: req.Status, Priority: req.Priority}
	if req.AssigneeID != nil {
		input.AssigneeSet = true
		if string(req.AssigneeID) != "null" {
			var value string
			if json.Unmarshal(req.AssigneeID, &value) != nil {
				response.WriteError(w, r, 422, "VALIDATION_ERROR", "Request validation failed", nil)
				return
			}
			input.AssigneeID = &value
		}
	}
	if req.DueDate != nil {
		input.DueDateSet = true
		if string(req.DueDate) != "null" {
			var value string
			if json.Unmarshal(req.DueDate, &value) != nil {
				response.WriteError(w, r, 422, "VALIDATION_ERROR", "Request validation failed", nil)
				return
			}
			input.DueDate = &value
		}
	}
	i, err := h.service.Update(r.Context(), actor(r), projectID, issueID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.Write(w, 200, Present(i))
}
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	projectID, issueID, err := params(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Issue ID is invalid", nil)
		return
	}
	var req StatusRequest
	if err = decode(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	if err = validateRequest(req); err != nil {
		response.WriteError(w, r, 422, "VALIDATION_ERROR", "Request validation failed", nil)
		return
	}
	i, err := h.service.Move(r.Context(), actor(r), projectID, issueID, req.Status, 0)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.Write(w, 200, Present(i))
}
func (h *Handler) Position(w http.ResponseWriter, r *http.Request) {
	projectID, issueID, err := params(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Issue ID is invalid", nil)
		return
	}
	var req PositionRequest
	if err = decode(r, &req); err != nil {
		response.WriteError(w, r, 400, "MALFORMED_JSON", "Request body must be valid JSON", nil)
		return
	}
	if err = validateRequest(req); err != nil {
		response.WriteError(w, r, 422, "VALIDATION_ERROR", "Request validation failed", nil)
		return
	}
	i, err := h.service.Move(r.Context(), actor(r), projectID, issueID, req.Status, req.Position)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.Write(w, 200, Present(i))
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, issueID, err := params(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Issue ID is invalid", nil)
		return
	}
	if err = h.service.Delete(r.Context(), actor(r), projectID, issueID); err != nil {
		h.writeError(w, r, err)
		return
	}
	response.WriteNoContent(w)
}
func (h *Handler) Activities(w http.ResponseWriter, r *http.Request) {
	projectID, issueID, err := params(r)
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Issue ID is invalid", nil)
		return
	}
	items, err := h.service.Activities(r.Context(), actor(r), projectID, issueID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]any, 0, len(items))
	for _, a := range items {
		out = append(out, PresentActivity(a))
	}
	response.WriteCollection(w, 200, out, response.PaginationMeta(1, len(out), len(out)))
}
func parseInt(value string) int {
	var n int
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
		if n > 100000 {
			return 100000
		}
	}
	return n
}
func csv(value string) []string {
	if value == "" {
		return nil
	}
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
func validateRequest(value any) error {
	if err := validation.Validator.Struct(value); err != nil {
		return err
	}
	return nil
}
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, app.ErrValidation):
		response.WriteError(w, r, 422, "VALIDATION_ERROR", "Request validation failed", nil)
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
