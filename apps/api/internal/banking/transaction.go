package banking

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"time"
)

func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		slog.Error("bank transaction rollback failed")
	}
}
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// lockBankHousehold serializes Link lifecycle changes before connection/attempt locks.
func lockBankHousehold(ctx context.Context, tx pgx.Tx, id string) error {
	var locked string
	return tx.QueryRow(ctx, `SELECT id FROM households WHERE id=$1 FOR UPDATE`, id).Scan(&locked)
}
