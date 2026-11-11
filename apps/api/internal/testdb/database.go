// Package testdb creates an isolated PostgreSQL schema for integration tests.
package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"bankpiggy/api/internal/store"
	"github.com/jackc/pgx/v5"
)

// Open creates a disposable schema; absent TEST_DATABASE_URL skips the test.
func Open(t *testing.T) *store.Store {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is required for real PostgreSQL integration tests")
	}
	ctx := context.Background()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := connection.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := connection.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := connection.Exec(ctx, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	database, err := store.Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Pool.Close)
	return database
}
