package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Basith-08/tracklume-api/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestPostgresConstraintsAndIssueSequence(t *testing.T) {
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
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	owner, member := uuid.New(), uuid.New()
	if _, err = tx.Exec(ctx, `INSERT INTO users(id,name,email,password_hash) VALUES($1,'Integration Owner',$2,'hash'),($3,'Integration Member',$4,'hash')`, owner, owner.String()+"@test.local", member, member.String()+"@test.local"); err != nil {
		t.Fatal(err)
	}
	projectID := uuid.New()
	if _, err = tx.Exec(ctx, `INSERT INTO projects(id,name,key,owner_id) VALUES($1,'Integration','ITG',$2)`, projectID, owner); err != nil {
		t.Fatal(err)
	}
	var role string
	if err = tx.QueryRow(ctx, `SELECT role FROM project_members WHERE project_id=$1 AND user_id=$2`, projectID, owner).Scan(&role); err != nil || role != "owner" {
		t.Fatalf("owner membership missing: %v/%s", err, role)
	}
	if _, err = tx.Exec(ctx, `SAVEPOINT duplicate_email`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO users(id,name,email,password_hash) VALUES($1,'Duplicate',$2,'hash')`, uuid.New(), owner.String()+"@test.local"); err == nil {
		t.Fatal("duplicate email accepted")
	}
	if _, err = tx.Exec(ctx, `ROLLBACK TO SAVEPOINT duplicate_email`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `RELEASE SAVEPOINT duplicate_email`); err != nil {
		t.Fatal(err)
	}
	var first, second int64
	if err = tx.QueryRow(ctx, `UPDATE project_issue_counters SET next_sequence=next_sequence+1 WHERE project_id=$1 RETURNING next_sequence-1`, projectID).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `UPDATE project_issue_counters SET next_sequence=next_sequence+1 WHERE project_id=$1 RETURNING next_sequence-1`, projectID).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if first != 1 || second != 2 {
		t.Fatalf("sequence = %d,%d", first, second)
	}
	if !errors.Is(tx.Rollback(ctx), nil) && !errors.Is(tx.Rollback(ctx), pgx.ErrTxClosed) {
		t.Fatal("rollback failed")
	}
}
