//go:build integration

package migrations

import (
	"context"
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"os"
	"testing"
	"time"
)

func TestRealDependencies(t *testing.T) {
	databaseURL, redisURL := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_REDIS_URL")
	if databaseURL == "" || redisURL == "" {
		t.Fatal("TEST_DATABASE_URL and TEST_REDIS_URL required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// The migration must be idempotent on a running stack.
	if err := Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	var version string
	if err := db.QueryRowContext(ctx, "SELECT extversion FROM pg_extension WHERE extname='vector'").Scan(&version); err != nil {
		t.Fatal(err)
	}
	var distance float64
	if err := db.QueryRowContext(ctx, "SELECT '[1,0,0]'::vector <=> '[0,1,0]'::vector").Scan(&distance); err != nil || distance != 1 {
		t.Fatalf("vector cosine: %v, %v", distance, err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var id string
	if err := tx.QueryRowContext(ctx, "INSERT INTO users DEFAULT VALUES RETURNING id").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO profiles(user_id,display_name) VALUES ($1,'Integration user')", id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE users SET status='invented' WHERE id=$1", id); err == nil {
		t.Fatal("invalid account status accepted")
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(opts)
	defer client.Close()
	key := "integration:" + id
	if err := client.Set(ctx, key, "present", time.Second).Err(); err != nil {
		t.Fatal(err)
	}
	defer client.Del(context.Background(), key)
	ttl, err := client.PTTL(ctx, key).Result()
	if err != nil || ttl <= 0 || ttl > time.Second {
		t.Fatalf("unexpected presence TTL %v: %v", ttl, err)
	}
	t.Log("PostgreSQL migration re-run, constraints, pgvector cosine, Redis TTL passed")
}
