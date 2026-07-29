package main

import (
	"context"
	"fmt"
	"github.com/Basith-08/tracklume-api/internal/database"
	"log"
	"os"
)

func main() {
	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}
	if direction != "up" && direction != "down" {
		log.Fatal("usage: go run ./cmd/migrate [up|down]")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", env("POSTGRES_USER", "tracklume"), os.Getenv("POSTGRES_PASSWORD"), env("DB_HOST", "localhost"), env("DB_PORT", "5432"), env("POSTGRES_DB", "tracklume"), env("DB_SSLMODE", "disable"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*1000000000)
	defer cancel()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	if err = database.ApplyMigration(ctx, pool, "migrations", direction); err != nil {
		log.Fatal(err)
	}
	log.Printf("migration %s complete", direction)
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
