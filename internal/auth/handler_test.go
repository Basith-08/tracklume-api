package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Basith-08/tracklume-api/internal/middleware"
	"github.com/Basith-08/tracklume-api/internal/security"
)

func TestRegisterRejectsMalformedJSON(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString("{"))
	res := httptest.NewRecorder()
	handler.Register(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got status %d", res.Code)
	}
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	manager := security.NewTokenManager("test-secret", 0)
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	res := httptest.NewRecorder()
	middleware.Authenticate(manager, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("authenticated handler called") })).ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d", res.Code)
	}
}
