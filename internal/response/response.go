package response

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type ErrorBody struct {
	Code      string              `json:"code"`
	Message   string              `json:"message"`
	Fields    map[string][]string `json:"fields,omitempty"`
	RequestID string              `json:"request_id"`
}

type errorEnvelope struct {
	Error ErrorBody `json:"error"`
}
type dataEnvelope struct {
	Data any `json:"data"`
}

type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type collectionEnvelope struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

func Write(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(dataEnvelope{Data: data})
}

func WriteCollection(w http.ResponseWriter, status int, data any, meta Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(collectionEnvelope{Data: data, Meta: meta})
}

func WriteNoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string, fields map[string][]string) {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: ErrorBody{Code: code, Message: message, Fields: fields, RequestID: requestID}})
}

func NormalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func PaginationMeta(page, perPage, total int) Meta {
	pages := 0
	if total > 0 {
		pages = (total + perPage - 1) / perPage
	}
	return Meta{Page: page, PerPage: perPage, Total: total, TotalPages: pages}
}
