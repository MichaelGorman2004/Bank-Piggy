package banking

import (
	"errors"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestInvestmentMatchRequiresExplicitInvestmentPlanAndKeepsManualProgress(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, true)
	saveServiceActivity(t, s, identity, connection, "funding", connection.Accounts[0].ID, "2026-11-03", -101, func(r *budget.ActivityRecord) { r.ActivitySubtype = "contribution" })
	request := budget.ActivityReviewRequest{ExpectedRevision: 1, ExpectedPlanRevision: 1, Category: ptr(budget.ActivityCategory("contribution")), GoalID: ptr("retirement")}
	if _, err := s.ReviewActivity(t.Context(), identity, "funding", request); !errors.Is(err, ErrInvalid) {
		t.Fatal("editable retirement name was treated as an investment type", err)
	}
	plan := updateMatchPlan(t, s, identity, func(h *budget.Household) { h.Goals[1].Purpose = "investment" })
	request.ExpectedPlanRevision = plan.Revision
	got, err := s.ReviewActivity(t.Context(), identity, "funding", request)
	if err != nil || got.Allocation == nil || got.Allocation.FirstMemberCents != -51 || got.GoalID == nil {
		t.Fatal("investment goal match did not retain exact raw shares", err)
	}
	requireUnchangedMatchPlan(t, s, identity, plan)
	second := serviceActivityMonth(t, s, identity, "member-2", "")
	if second.Totals.NetInvestmentContributionsCents != 50 || second.Comparison.InvestmentGoals[0].NetContributionsCents != 50 {
		t.Fatal("investment goal allocation did not reach the selected member")
	}
}

func requireUnchangedMatchPlan(t *testing.T, s *Service, identity Identity, before budget.Household) {
	t.Helper()
	after, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil || after.Revision != before.Revision || after.Goals[0].SavingsTarget.SavedAmountCents != before.Goals[0].SavingsTarget.SavedAmountCents {
		t.Fatal("matching changed plan revision or manual progress", err)
	}
}

func TestPlanEditHoldingHouseholdLockFencesConcurrentMatch(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, false)
	saveServiceActivity(t, s, identity, connection, "paid", connection.Accounts[0].ID, "2026-11-03", 101, nil)
	tx, err := s.db.Pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer rollback(tx)
	if _, err := tx.Exec(t.Context(), `UPDATE households SET revision=revision+1,plan=jsonb_set(plan,'{revision}',to_jsonb(revision+1)) WHERE id=$1`, identity.HouseholdID); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		close(started)
		_, err := s.ReviewActivity(t.Context(), identity, "paid", expenseReview(1, 1))
		result <- err
	}()
	<-started
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, ErrConflict) {
		t.Fatal("match ignored a concurrent plan update", err)
	}
	requireServiceReviewRow(t, s, "paid", "", 1)
}
