package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Basith-08/tracklume-api/internal/admin"
	app "github.com/Basith-08/tracklume-api/internal/apperror"
	"github.com/Basith-08/tracklume-api/internal/auth"
	"github.com/Basith-08/tracklume-api/internal/database"
	"github.com/Basith-08/tracklume-api/internal/security"
	"github.com/google/uuid"
)

func TestPlatformAdminLifecycle(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	superadminID := uuid.New()
	userID := uuid.New()
	passwordHash, err := security.HashPassword("Password123!")
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO users(id,name,email,password_hash,platform_role)
VALUES ($1,'Integration Superadmin',$2,$3,'superadmin'),($4,'Integration User',$5,$3,'user')`,
		superadminID, superadminID.String()+"@admin.test", passwordHash, userID, userID.String()+"@admin.test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id IN ($1,$2)`, superadminID, userID)
	}()

	authRepo := auth.NewRepository(pool)
	authService := auth.NewService(authRepo, security.NewTokenManager("integration-test-secret", time.Hour))
	login, err := authService.Login(ctx, auth.LoginRequest{Email: userID.String() + "@admin.test", Password: "Password123!"})
	if err != nil || login.User.LastLoginAt == nil {
		t.Fatalf("login/last_login_at failed: %v", err)
	}

	adminService := admin.NewService(admin.NewRepository(pool))
	overview, err := adminService.Overview(ctx, superadminID)
	if err != nil || overview.TotalUsers < 2 || overview.ActiveUsers < 2 {
		t.Fatalf("overview = %+v, err = %v", overview, err)
	}
	users, total, err := adminService.ListUsers(ctx, superadminID, admin.Filter{Search: "Integration User", Status: "active", Page: 1, PerPage: 20})
	if err != nil || total != 1 || len(users) != 1 || users[0].ID != userID {
		t.Fatalf("users=%+v total=%d err=%v", users, total, err)
	}
	if _, _, err = adminService.ListUsers(ctx, userID, admin.Filter{Page: 1, PerPage: 20}); !errors.Is(err, app.ErrForbidden) {
		t.Fatalf("regular user admin access error = %v", err)
	}

	updated, err := adminService.UpdateStatus(ctx, superadminID, userID, admin.UpdateStatusRequest{IsActive: false, Reason: "Support review"})
	if err != nil || updated.IsActive || updated.DeactivationReason == nil {
		t.Fatalf("deactivation = %+v, err = %v", updated, err)
	}
	active, err := authRepo.IsActive(ctx, userID)
	if err != nil || active {
		t.Fatalf("active status = %v, err = %v", active, err)
	}
	if _, err = authService.Login(ctx, auth.LoginRequest{Email: userID.String() + "@admin.test", Password: "Password123!"}); !errors.Is(err, app.ErrInactive) {
		t.Fatalf("inactive login error = %v", err)
	}

	updated, err = adminService.UpdateStatus(ctx, superadminID, userID, admin.UpdateStatusRequest{IsActive: true})
	if err != nil || !updated.IsActive || updated.DeactivatedAt != nil {
		t.Fatalf("reactivation = %+v, err = %v", updated, err)
	}

	if err = adminService.DeleteUser(ctx, superadminID, userID); err != nil {
		t.Fatalf("soft delete = %v", err)
	}
	deleted, deletedTotal, err := adminService.ListUsers(ctx, superadminID, admin.Filter{Status: "deleted", Page: 1, PerPage: 20})
	if err != nil || deletedTotal != 1 || len(deleted) != 1 || deleted[0].DeletedAt == nil {
		t.Fatalf("deleted users=%+v total=%d err=%v", deleted, deletedTotal, err)
	}
	if _, err = authService.Login(ctx, auth.LoginRequest{Email: userID.String() + "@admin.test", Password: "Password123!"}); !errors.Is(err, app.ErrUnauthorized) {
		t.Fatalf("deleted login error = %v", err)
	}
	restored, err := adminService.RestoreUser(ctx, superadminID, userID)
	if err != nil || !restored.IsActive || restored.DeletedAt != nil {
		t.Fatalf("restore = %+v, err = %v", restored, err)
	}
}
