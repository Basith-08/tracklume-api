package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/Basith-08/tracklume-api/internal/admin"
	"github.com/Basith-08/tracklume-api/internal/database"
	"github.com/Basith-08/tracklume-api/internal/security"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "create" {
		log.Fatal("usage: ADMIN_BOOTSTRAP_PASSWORD='...' go run ./cmd/admin create --email admin@example.com --name Admin")
	}
	flags := flag.NewFlagSet("create", flag.ExitOnError)
	email := flags.String("email", "", "superadmin email")
	name := flags.String("name", "", "superadmin display name")
	_ = flags.Parse(os.Args[2:])
	password := os.Getenv("ADMIN_BOOTSTRAP_PASSWORD")
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*email) == "" || len(password) < 8 {
		log.Fatal("name and email are required, and ADMIN_BOOTSTRAP_PASSWORD must contain at least 8 characters")
	}
	if _, err := mail.ParseAddress(*email); err != nil {
		log.Fatal("email is invalid")
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		log.Fatal(err)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", env("POSTGRES_USER", "tracklume"), os.Getenv("POSTGRES_PASSWORD"), env("DB_HOST", "localhost"), env("DB_PORT", "5432"), env("POSTGRES_DB", "tracklume"), env("DB_SSLMODE", "disable"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	user, err := admin.NewRepository(pool).BootstrapSuperadmin(ctx, strings.TrimSpace(*name), strings.TrimSpace(*email), hash)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("superadmin ready: %s", user.Email)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
