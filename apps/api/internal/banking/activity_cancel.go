package banking

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func cancelActivityRecords(ctx context.Context, tx pgx.Tx, job activityWork, batch activityBatch) error {
	ids, err := activityCancellationIDs(ctx, tx, job, batch)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := invalidateActivityPair(ctx, tx, id); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE bank_activity_records SET cancelled=true,review_category=NULL,
 expense_id=NULL,goal_id=NULL,allocation=NULL,revision=revision+1
 WHERE id=ANY($1) AND (NOT cancelled OR review_category IS NOT NULL OR allocation IS NOT NULL)`, ids)
	return err
}

func activityCancellationIDs(ctx context.Context, tx pgx.Tx, job activityWork, batch activityBatch) ([]string, error) {
	providerIDs := make([]string, 0, len(batch.Records))
	for _, record := range batch.Records {
		providerIDs = append(providerIDs, record.ProviderID)
	}
	rows, err := tx.Query(ctx, `SELECT r.id FROM bank_activity_records r WHERE r.connection_id=$1 AND (
 r.cancelled OR r.provider_id=ANY($2) OR EXISTS(SELECT 1 FROM bank_activity_cancellations c
 WHERE c.connection_id=r.connection_id AND c.provider_id=r.provider_id)
 OR ($3='investment' AND r.posted_on BETWEEN $4::date AND $5::date AND NOT(r.provider_id=ANY($6))))
 ORDER BY r.id`, job.ID, batch.Removed, job.Kind, batch.Start, batch.End, providerIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
