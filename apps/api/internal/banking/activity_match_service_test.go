package banking

import (
	"errors"
	"testing"

	"bankpiggy/api/internal/budget"
)

func updateMatchPlan(t *testing.T, s *Service, identity Identity, change func(*budget.Household)) budget.Household {
	t.Helper()
	plan, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	change(&plan)
	if err := budget.Validate(plan); err != nil {
		t.Fatal(err)
	}
	plan, err = s.db.Update(t.Context(), identity.HouseholdID, plan)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func expenseReview(revision, planRevision int64) budget.ActivityReviewRequest {
	return budget.ActivityReviewRequest{ExpectedRevision: revision, ExpectedPlanRevision: planRevision, Category: ptr(budget.ActivityCategory("spending")), ExpenseID: ptr("rent")}
}

func TestExpenseMatchPersistsExactSplitAcrossPlanEditsAndDeletion(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, false)
	saveServiceActivity(t, s, identity, connection, "paid", connection.Accounts[0].ID, "2026-11-03", 101, nil)
	got, err := s.ReviewActivity(t.Context(), identity, "paid", expenseReview(1, 1))
	if err != nil || got.Allocation == nil || got.Allocation.FirstMemberCents != 67 || got.Ownership != "member-1" {
		t.Fatal("salary expense match was not stored separately from payer", err)
	}
	if got.ScopeAmountCents != nil {
		t.Fatal("scope-free review response exposed the full payer amount as an allocated share")
	}
	plan := updateMatchPlan(t, s, identity, func(h *budget.Household) { h.Members[0].AnnualBaseSalaryCents = 0 })
	got, err = s.ReviewActivity(t.Context(), identity, "paid", expenseReview(2, plan.Revision))
	if err != nil || got.Allocation.FirstMemberCents != 67 {
		t.Fatal("saving the same match silently recalculated recorded shares", err)
	}
	second := serviceActivityMonth(t, s, identity, "member-2", "")
	requireCrossPayerMatch(t, second)
	updateMatchPlan(t, s, identity, func(h *budget.Household) { h.Expenses = h.Expenses[1:] })
	second = serviceActivityMonth(t, s, identity, "member-2", "")
	requireDeletedMatch(t, second)
}

func requireCrossPayerMatch(t *testing.T, second budget.ActivityResponse) {
	t.Helper()
	if len(second.Records) != 1 || second.Totals.SpendingCents != 34 || *second.Records[0].ScopeAmountCents != 34 {
		t.Fatal("payer scope filtered another member's allocated expense")
	}
}

func requireDeletedMatch(t *testing.T, second budget.ActivityResponse) {
	t.Helper()
	if second.Comparison.UnmatchedSpendingCents != 34 || second.Records[0].ExpenseID == nil || second.Records[0].Allocation == nil {
		t.Fatal("deleting a plan target removed actual money or frozen allocation")
	}
}

func TestClearingAndRematchingExplicitlyRecalculatesExpenseShare(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, false)
	saveServiceActivity(t, s, identity, connection, "refund", connection.Accounts[0].ID, "2026-11-03", -101, nil)
	got, err := s.ReviewActivity(t.Context(), identity, "refund", expenseReview(1, 1))
	if err != nil || got.Allocation.FirstMemberCents != -67 {
		t.Fatal("refund did not reverse the exact positive split", err)
	}
	plan := updateMatchPlan(t, s, identity, func(h *budget.Household) { h.Members[0].AnnualBaseSalaryCents = 0 })
	clear := expenseReview(2, plan.Revision)
	clear.ExpenseID = nil
	got, err = s.ReviewActivity(t.Context(), identity, "refund", clear)
	if err != nil || got.ExpenseID != nil || got.Allocation != nil {
		t.Fatal("clearing match retained its allocation", err)
	}
	got, err = s.ReviewActivity(t.Context(), identity, "refund", expenseReview(3, plan.Revision))
	if err != nil || got.Allocation.FirstMemberCents != 0 {
		t.Fatal("explicit rematch did not use the current split", err)
	}
}

func TestExpenseMatchRejectsStalePlansAndUnknownTargetsWithoutWrites(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, false)
	saveServiceActivity(t, s, identity, connection, "paid", connection.Accounts[0].ID, "2026-11-03", 101, nil)
	updateMatchPlan(t, s, identity, func(h *budget.Household) { h.Expenses[0].Name = "Updated rent" })
	if _, err := s.ReviewActivity(t.Context(), identity, "paid", expenseReview(1, 1)); !errors.Is(err, ErrConflict) {
		t.Fatal("stale plan accepted a match", err)
	}
	request := expenseReview(1, 2)
	request.ExpenseID = ptr("other-household-expense")
	if _, err := s.ReviewActivity(t.Context(), identity, "paid", request); !errors.Is(err, ErrNotFound) {
		t.Fatal("foreign target ID was accepted", err)
	}
	requireServiceReviewRow(t, s, "paid", "", 1)
	if got := serviceActivityMonth(t, s, identity, "household", "").Records[0]; got.ExpenseID != nil || got.Allocation != nil {
		t.Fatal("rejected match partially wrote review data")
	}
}

func TestExpenseMatchRejectsConflictingAndUnsupportedReviewTargets(t *testing.T) {
	cases := []struct {
		name   string
		change func(*budget.ActivityReviewRequest)
	}{
		{"both targets", func(r *budget.ActivityReviewRequest) { r.GoalID = ptr("retirement") }},
		{"income match", func(r *budget.ActivityReviewRequest) { r.Category = ptr(budget.ActivityCategory("income")) }},
		{"automatic classification", func(r *budget.ActivityReviewRequest) { r.Category = nil }},
		{"transfer match", func(r *budget.ActivityReviewRequest) {
			r.Category = ptr(budget.ActivityCategory("transfer"))
			r.TransferPeerID = ptr("peer")
			r.ExpectedPeerRevision = ptr(int64(1))
		}},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			s, identity, connection := serviceActivityFixture(t, false)
			saveServiceActivity(t, s, identity, connection, "paid", connection.Accounts[0].ID, "2026-11-03", 101, nil)
			request := expenseReview(1, 1)
			item.change(&request)
			if _, err := s.ReviewActivity(t.Context(), identity, "paid", request); !errors.Is(err, ErrInvalid) {
				t.Fatal("invalid target combination was accepted", err)
			}
			requireServiceReviewRow(t, s, "paid", "", 1)
		})
	}
}
