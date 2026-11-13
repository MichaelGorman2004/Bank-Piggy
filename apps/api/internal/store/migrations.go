package store

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed accounts_migration.sql
var accountsMigration string

//go:embed banking_schema.sql
var bankingSchema string

// migrateSchema atomically creates tables and upgrades saved account snapshots.
func migrateSchema(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if _, err = tx.Exec(ctx, schema); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, accountsMigration); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, bankingSchema); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
