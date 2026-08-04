package dashboard

import (
	"errors"
	app "github.com/Basith-08/tracklume-api/internal/apperror"
	"github.com/Basith-08/tracklume-api/internal/middleware"
	"github.com/Basith-08/tracklume-api/internal/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		response.WriteError(w, r, 400, "INVALID_ID", "Project ID is invalid", nil)
		return
	}
	userID, _ := middleware.UserIDFromContext(r.Context())
	result, err := h.service.Get(r.Context(), userID, projectID)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrNotFound):
			response.WriteError(w, r, 404, "NOT_FOUND", "Resource not found", nil)
		case errors.Is(err, app.ErrForbidden):
			response.WriteError(w, r, 403, "FORBIDDEN", "You do not have permission for this project", nil)
		default:
			response.WriteError(w, r, 500, "INTERNAL_ERROR", "An unexpected error occurred", nil)
		}
		return
	}
	response.Write(w, 200, result)
}
