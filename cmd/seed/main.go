package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Basith-08/tracklume-api/internal/database"
	"github.com/Basith-08/tracklume-api/internal/security"
	"github.com/google/uuid"
)

func main() {
	if os.Getenv("APP_ENV") == "production" && os.Getenv("ALLOW_DEMO_SEED") != "true" {
		log.Fatal("refusing to seed demo data in production; set ALLOW_DEMO_SEED=true for an explicit one-time seed")
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
	ownerHash, err := security.HashPassword("Password123!")
	if err != nil {
		log.Fatal(err)
	}
	memberHash, err := security.HashPassword("Password123!")
	if err != nil {
		log.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx)
	ownerID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("tracklume-owner"))
	memberID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("tracklume-member"))
	if _, err = tx.Exec(ctx, `INSERT INTO users(id,name,email,password_hash) VALUES($1,'Tracklume Owner','owner@tracklume.local',$2) ON CONFLICT(email) DO UPDATE SET name=EXCLUDED.name,password_hash=EXCLUDED.password_hash,is_active=true,deactivated_at=NULL,deactivation_reason=NULL,deleted_at=NULL`, ownerID, ownerHash); err != nil {
		log.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO users(id,name,email,password_hash) VALUES($1,'Tracklume Member','member@tracklume.local',$2) ON CONFLICT(email) DO UPDATE SET name=EXCLUDED.name,password_hash=EXCLUDED.password_hash,is_active=true,deactivated_at=NULL,deactivation_reason=NULL,deleted_at=NULL`, memberID, memberHash); err != nil {
		log.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `SELECT id FROM users WHERE email='owner@tracklume.local'`).Scan(&ownerID); err != nil {
		log.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `SELECT id FROM users WHERE email='member@tracklume.local'`).Scan(&memberID); err != nil {
		log.Fatal(err)
	}
	projectID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("tracklume-demo-project"))
	if _, err = tx.Exec(ctx, `INSERT INTO projects(id,name,key,description,owner_id) VALUES($1,'Tracklume Demo','DEMO','Local development demo project',$2) ON CONFLICT(key) DO UPDATE SET name=EXCLUDED.name,description=EXCLUDED.description RETURNING id`, projectID, ownerID); err != nil {
		log.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `SELECT id FROM projects WHERE key='DEMO'`).Scan(&projectID); err != nil {
		log.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO project_members(project_id,user_id,role) VALUES($1,$2,'member') ON CONFLICT DO NOTHING`, projectID, memberID); err != nil {
		log.Fatal(err)
	}
	types := []string{"task", "bug", "feature"}
	statuses := []string{"backlog", "todo", "in_progress", "done", "cancelled"}
	priorities := []string{"low", "medium", "high", "urgent"}
	for n := 1; n <= 10; n++ {
		id := uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("tracklume-demo-issue-%d", n)))
		assignee := any(nil)
		if n%2 == 0 {
			assignee = memberID
		}
		due := any(nil)
		if n%3 != 0 {
			due = time.Now().UTC().AddDate(0, 0, n-5).Format("2006-01-02")
		}
		_, err = tx.Exec(ctx, `INSERT INTO issues(id,project_id,sequence_number,identifier,title,description,type,status,priority,assignee_id,reporter_id,due_date,position) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT(id) DO NOTHING`, id, projectID, n, fmt.Sprintf("DEMO-%d", n), fmt.Sprintf("Demo issue %d", n), "Seeded local demo issue", types[n%len(types)], statuses[n%len(statuses)], priorities[n%len(priorities)], assignee, ownerID, due, n)
		if err != nil {
			log.Fatal(err)
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE project_issue_counters SET next_sequence=GREATEST(next_sequence,11) WHERE project_id=$1`, projectID); err != nil {
		log.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("idempotent demo seed complete")
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
