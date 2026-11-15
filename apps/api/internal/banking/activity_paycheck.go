package banking

import (
	"context"
	"encoding/json"
	"time"

	"bankpiggy/api/internal/activity"
	"bankpiggy/api/internal/budget"
)

// ApplyPaycheck recomputes the exact suggestion under the household lock before confirmation.
// It changes only the selected member's net pay and pay calendar, with plan revision protection.
func (s *Service) ApplyPaycheck(ctx context.Context, identity Identity, id string, request budget.ApplyPaycheckRequest) (budget.Household, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return budget.Household{}, err
	}
	defer rollback(tx)
	ledger, err := s.readActivityLedger(ctx, tx, identity, true)
	if err != nil {
		return ledger.plan, err
	}
	if request.ExpectedPlanRevision != ledger.plan.Revision {
		return ledger.plan, ErrConflict
	}
	if !budget.HasMember(ledger.plan, request.MemberID) {
		return ledger.plan, ErrInvalid
	}
	projection, err := activity.Project(ledger.plan, ledger.records, ledger.sources, time.Now().UTC().Truncate(24*time.Hour), "household", time.Now())
	if err != nil {
		return ledger.plan, ErrInvalid
	}
	if !applyPaySuggestion(&ledger.plan, projection.Suggestions, id, request.MemberID) {
		return ledger.plan, ErrNotFound
	}
	if err := budget.Validate(ledger.plan); err != nil {
		return ledger.plan, ErrInvalid
	}
	ledger.plan.Revision++
	data, err := json.Marshal(ledger.plan)
	if err != nil {
		return ledger.plan, err
	}
	_, err = tx.Exec(ctx, `UPDATE households SET revision=$2,plan=$3,updated_at=now() WHERE id=$1`, identity.HouseholdID, ledger.plan.Revision, data)
	if err != nil {
		return ledger.plan, err
	}
	return ledger.plan, tx.Commit(ctx)
}

func applyPaySuggestion(plan *budget.Household, suggestions []budget.PaycheckSuggestion, id, memberID string) bool {
	for _, suggestion := range suggestions {
		if suggestion.ID != id {
			continue
		}
		for index := range plan.Members {
			if plan.Members[index].ID == memberID {
				plan.Members[index].MonthlyTakeHomeCents = suggestion.MonthlyTakeHomeCents
				plan.Members[index].PayFrequency = suggestion.Frequency
				plan.Members[index].PaySchedule = &suggestion.PaySchedule
				return true
			}
		}
	}
	return false
}
