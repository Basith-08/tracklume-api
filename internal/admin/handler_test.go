package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestUpdateStatusRejectsMalformedJSON(t *testing.T) {
	handler := NewHandler(nil)
	r := chi.NewRouter()
	r.Patch("/users/{userID}/status", handler.UpdateStatus)
	req := httptest.NewRequest(http.MethodPatch, "/users/00000000-0000-0000-0000-000000000001/status", bytes.NewBufferString("{"))
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got status %d", res.Code)
	}
}
