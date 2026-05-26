package db

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open returns a connection pool configured from environment variables.
//
// Recognised env vars:
//
//	DATABASE_URL  — full connection string (takes precedence)
//	DB_HOST       — defaults to "localhost"
//	DB_PORT       — defaults to "5432"
//	DB_USER       — defaults to "redgrave"
//	DB_PASSWORD   — defaults to "redgrave"
//	DB_NAME       — defaults to "redgrave"
//	DB_SSLMODE    — defaults to "disable"
func Open(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := envOrDefault("DB_HOST", "localhost")
		port := envOrDefault("DB_PORT", "5432")
		user := envOrDefault("DB_USER", "redgrave")
		password := envOrDefault("DB_PASSWORD", "redgrave")
		dbname := envOrDefault("DB_NAME", "redgrave")
		sslmode := envOrDefault("DB_SSLMODE", "disable")

		u := &url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(user, password),
			Host:   net.JoinHostPort(host, port),
			Path:   dbname,
		}
		q := u.Query()
		q.Set("sslmode", sslmode)
		u.RawQuery = q.Encode()
		dsn = u.String()
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
