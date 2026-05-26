package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// requiresDB skips the test when no PostgreSQL instance is reachable.
func requiresDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if os.Getenv("REDGRAVE_TEST_DB") == "" {
		t.Skip("REDGRAVE_TEST_DB not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := Open(ctx)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestOpen_WithURL(t *testing.T) {
	url := os.Getenv("REDGRAVE_TEST_DB_URL")
	if url == "" {
		t.Skip("REDGRAVE_TEST_DB_URL not set")
	}

	t.Setenv("DATABASE_URL", url)
	t.Setenv("DB_HOST", "should-be-ignored")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := Open(ctx)
	if err != nil {
		t.Fatalf("Open with DATABASE_URL: %v", err)
	}
	defer pool.Close()

	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("SELECT 1: %v", err)
	}
	if one != 1 {
		t.Fatalf("expected 1, got %d", one)
	}
}

func TestOpen_WithComponents(t *testing.T) {
	pool := requiresDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("SELECT 1: %v", err)
	}
	if one != 1 {
		t.Fatalf("expected 1, got %d", one)
	}
}

func TestOpen_InvalidHost(t *testing.T) {
	t.Setenv("DB_HOST", "255.255.255.255")
	t.Setenv("DB_PORT", "5432")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Open(ctx)
	if err == nil {
		t.Fatal("expected error for unreachable host")
	}
}
