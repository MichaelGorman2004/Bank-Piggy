package banking

import (
	"context"
	"encoding/json"
	"time"

	"bankpiggy/api/internal/activity"
	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

const maxSavedActivityRecords = 50000

type activityLedger struct {
	plan    budget.Household
	sources []budget.ActivitySource
	records []budget.ActivityRecord
}

// MonthlyActivity reads one consistent saved snapshot. It never contacts Plaid or changes the plan.
func (s *Service) MonthlyActivity(ctx context.Context, identity Identity, month time.Time, scope string) (budget.ActivityResponse, error) {
	tx, err := s.db.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return budget.ActivityResponse{}, err
	}
	defer rollback(tx)
	ledger, err := s.readActivityLedger(ctx, tx, identity, false)
	if err != nil {
		return budget.ActivityResponse{}, err
	}
	result, err := activity.Project(ledger.plan, ledger.records, ledger.sources, month, scope, time.Now())
	if err != nil {
		return result, ErrInvalid
	}
	result.Environment = s.config.Environment
	if !s.config.Enabled {
		result.Environment = "disabled"
	}
	return result, tx.Commit(ctx)
}

func (s *Service) readActivityLedger(ctx context.Context, tx pgx.Tx, identity Identity, lock bool) (activityLedger, error) {
	var ledger activityLedger
	query := `SELECT plan FROM households WHERE id=$1`
	if lock {
		query += ` FOR UPDATE`
	}
	var data []byte
	if err := tx.QueryRow(ctx, query, identity.HouseholdID).Scan(&data); err != nil {
		return ledger, notFound(err)
	}
	if err := json.Unmarshal(data, &ledger.plan); err != nil {
		return ledger, err
	}
	if !budget.HasMember(ledger.plan, identity.MemberID) {
		return ledger, ErrNotFound
	}
	var err error
	ledger.sources, err = s.readActivitySources(ctx, tx, identity)
	if err != nil {
		return ledger, err
	}
	ledger.records, err = s.readActivityRecords(ctx, tx, identity.HouseholdID)
	assignActivityOwnership(&ledger)
	return ledger, err
}

func assignActivityOwnership(ledger *activityLedger) {
	owners := map[string]string{}
	for _, source := range ledger.sources {
		for _, account := range source.Accounts {
			owners[account.AccountID] = account.Ownership
		}
	}
	for index := range ledger.records {
		ledger.records[index].Ownership = owners[ledger.records[index].AccountID]
	}
}

func (s *Service) readActivityRecords(ctx context.Context, tx pgx.Tx, household string) ([]budget.ActivityRecord, error) {
	rows, err := tx.Query(ctx, `SELECT r.id,r.connection_id,r.account_id,r.kind,r.posted_on,r.data,
 r.cancelled,r.review_category,r.transfer_peer_id,r.revision,COALESCE(a.snapshot->>'name','Account'),
 r.expense_id,r.goal_id,r.allocation
 FROM bank_activity_records r JOIN bank_connections c ON c.id=r.connection_id AND c.household_id=r.household_id
 JOIN bank_accounts a ON a.id=r.account_id AND a.connection_id=r.connection_id
 WHERE r.household_id=$1 AND c.environment=$2 AND c.state IN ('connected','delayed','reconnect') AND a.active
 ORDER BY r.id LIMIT $3`, household, s.config.Environment, maxSavedActivityRecords+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []budget.ActivityRecord{}
	for rows.Next() {
		record, err := scanActivityRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
		if len(records) > maxSavedActivityRecords {
			return nil, ErrLimit
		}
	}
	return records, rows.Err()
}

func scanActivityRecord(row pgx.Row) (budget.ActivityRecord, error) {
	var record budget.ActivityRecord
	var data []byte
	var category *budget.ActivityCategory
	var date time.Time
	var allocation []byte
	err := row.Scan(&record.ID, &record.ConnectionID, &record.AccountID, &record.Kind, &date, &data,
		&record.Cancelled, &category, &record.TransferPeerID, &record.Revision, &record.AccountName,
		&record.ExpenseID, &record.GoalID, &allocation)
	if err != nil {
		return record, err
	}
	var raw budget.ActivityRecord
	if err := json.Unmarshal(data, &raw); err != nil {
		return record, err
	}
	raw.ID, raw.ConnectionID, raw.AccountID, raw.Kind = record.ID, record.ConnectionID, record.AccountID, record.Kind
	raw.Date, raw.AccountName, raw.Revision = budget.CalendarDate(date.Format(time.DateOnly)), record.AccountName, record.Revision
	raw.Cancelled, raw.TransferPeerID, raw.IsReviewed = record.Cancelled, record.TransferPeerID, category != nil
	raw.ExpenseID, raw.GoalID = record.ExpenseID, record.GoalID
	raw.Allocation = nil
	if allocation != nil {
		if err := json.Unmarshal(allocation, &raw.Allocation); err != nil {
			return record, err
		}
	}
	// Stored facts have no selected scope; the monthly projection allocates this amount.
	raw.ScopeAmountCents = nil
	if category != nil {
		raw.Category = *category
	}
	return raw, nil
}
