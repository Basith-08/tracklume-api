package security

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestJWTCreateAndValidate(t *testing.T) {
	manager := NewTokenManager("a-long-test-secret-that-is-safe", time.Hour)
	id := uuid.New()
	token, expires, err := manager.Create(id, time.Now())
	if err != nil || expires != 3600 {
		t.Fatalf("create token: %v", err)
	}
	got, err := manager.Parse(token)
	if err != nil || got != id {
		t.Fatalf("parse token: got %s err %v", got, err)
	}
	if _, err = NewTokenManager("different-secret", time.Hour).Parse(token); err == nil {
		t.Fatal("token signed by another secret was accepted")
	}
}
