package banking

import (
	"context"
	"encoding/json"

	"bankpiggy/api/internal/activity"
	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

// ReviewActivity saves one explicit classification, or both sides of an investment transfer.
// The household lock serializes reviews with imports, consent changes, and plan confirmations.
func (s *Service) ReviewActivity(ctx context.Context, identity Identity, id string, request budget.ActivityReviewRequest) (budget.ActivityRecord, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return budget.ActivityRecord{}, err
	}
	defer rollback(tx)
	ledger, err := s.readActivityLedger(ctx, tx, identity, true)
	if err != nil {
		return budget.ActivityRecord{}, err
	}
	records := reconciledActivity(ledger)
	record, err := validateActivityReviewTarget(ledger, records[id], request)
	if err != nil {
		return record, err
	}
	if err := persistActivityReview(ctx, tx, identity.HouseholdID, record, records, request); err != nil {
		return record, err
	}
	ledger, err = s.readActivityLedger(ctx, tx, identity, false)
	if err != nil {
		return record, err
	}
	result, exists := reconciledActivity(ledger)[id]
	if !exists {
		return record, ErrNotFound
	}
	return result, tx.Commit(ctx)
}

func validateActivityReviewTarget(ledger activityLedger, record budget.ActivityRecord, request budget.ActivityReviewRequest) (budget.ActivityRecord, error) {
	if _, included := activity.IncludedAccounts(ledger.sources)[record.AccountID]; !included {
		return record, ErrNotFound
	}
	if request.ExpectedPlanRevision != ledger.plan.Revision {
		return record, ErrConflict
	}
	if err := validateActivityReview(record, request); err != nil {
		return record, err
	}
	allocation, err := activityMatchAllocation(ledger.plan, record, request)
	record.Allocation = allocation
	return record, err
}

func reconciledActivity(ledger activityLedger) map[string]budget.ActivityRecord {
	result := map[string]budget.ActivityRecord{}
	for _, record := range activity.Reconcile(ledger.records) {
		result[record.ID] = record
	}
	return result
}

func validateActivityReview(record budget.ActivityRecord, request budget.ActivityReviewRequest) error {
	if request.ExpectedRevision != record.Revision {
		return ErrConflict
	}
	if request.Category != nil && !activity.ValidReview(record, *request.Category) {
		return ErrInvalid
	}
	if request.TransferPeerID == nil {
		if request.ExpectedPeerRevision != nil {
			return ErrInvalid
		}
		return nil
	}
	if request.Category == nil || *request.Category != "transfer" || request.ExpectedPeerRevision == nil {
		return ErrInvalid
	}
	return nil
}

func persistActivityReview(ctx context.Context, tx pgx.Tx, household string, record budget.ActivityRecord, records map[string]budget.ActivityRecord, request budget.ActivityReviewRequest) error {
	if request.TransferPeerID != nil {
		if err := validateActivityPartner(record, records, request); err != nil {
			return err
		}
	}
	if record.TransferPeerID != nil && (request.TransferPeerID == nil || *request.TransferPeerID != *record.TransferPeerID) {
		if err := clearActivityPartner(ctx, tx, household, record); err != nil {
			return err
		}
	}
	err := saveActivityReviewRow(ctx, tx, household, record, request)
	if err != nil || request.TransferPeerID == nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE bank_activity_records SET review_category='transfer',transfer_peer_id=$3,
 expense_id=NULL,goal_id=NULL,allocation=NULL,revision=revision+1
 WHERE household_id=$1 AND id=$2`, household, *request.TransferPeerID, record.ID)
	return err
}

func saveActivityReviewRow(ctx context.Context, tx pgx.Tx, household string, record budget.ActivityRecord, request budget.ActivityReviewRequest) error {
	var allocation []byte
	if record.Allocation != nil {
		var err error
		allocation, err = json.Marshal(record.Allocation)
		if err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `UPDATE bank_activity_records SET review_category=$3,transfer_peer_id=$4,
 expense_id=$5,goal_id=$6,allocation=$7,revision=revision+1 WHERE household_id=$1 AND id=$2`,
		household, record.ID, request.Category, request.TransferPeerID, request.ExpenseID, request.GoalID, allocation)
	return err
}

func validateActivityPartner(record budget.ActivityRecord, records map[string]budget.ActivityRecord, request budget.ActivityReviewRequest) error {
	peer, exists := records[*request.TransferPeerID]
	if !exists {
		return ErrNotFound
	}
	if peer.Revision != *request.ExpectedPeerRevision {
		return ErrConflict
	}
	if !activity.ValidTransferPair(record, peer) {
		return ErrInvalid
	}
	if peer.TransferPeerID != nil && *peer.TransferPeerID != record.ID {
		return ErrConflict
	}
	return nil
}

func clearActivityPartner(ctx context.Context, tx pgx.Tx, household string, record budget.ActivityRecord) error {
	_, err := tx.Exec(ctx, `UPDATE bank_activity_records SET review_category=NULL,transfer_peer_id=NULL,revision=revision+1
 WHERE household_id=$1 AND id=$2 AND transfer_peer_id=$3`, household, *record.TransferPeerID, record.ID)
	return err
}
