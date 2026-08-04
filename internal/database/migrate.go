package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ApplyMigration(ctx context.Context, pool *pgxpool.Pool, dir string, direction string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	if _, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version BIGINT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	versions := make([]int, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) != 2 {
			continue
		}
		version, parseErr := strconv.Atoi(parts[0])
		if parseErr == nil && strings.HasSuffix(entry.Name(), "."+direction+".sql") {
			versions = append(versions, version)
		}
	}
	sort.Ints(versions)
	if direction == "down" {
		sort.Sort(sort.Reverse(sort.IntSlice(versions)))
	}
	for _, version := range versions {
		var applied bool
		err = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version=$1)`, version).Scan(&applied)
		if err != nil {
			return err
		}
		if (direction == "up" && applied) || (direction == "down" && !applied) {
			continue
		}
		file := filepath.Join(dir, fmt.Sprintf("%03d_init.%s.sql", version, direction))
		sql, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %d: %w", version, err)
		}
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, string(sql)); err == nil {
			if direction == "up" {
				_, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, version)
			} else {
				_, err = tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version=$1`, version)
			}
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %d: %w", version, err)
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}
