package banking

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
)

func servicePaycheckFixture(t *testing.T) (*Service, Identity, budget.PaycheckSuggestion, budget.Household) {
	t.Helper()
	s, identity, connection := serviceActivityFixture(t, false)
	today := time.Now().UTC()
	for index := 0; index < 4; index++ {
		date := today.AddDate(0, 0, -7*(3-index)).Format(time.DateOnly)
		saveServiceActivity(t, s, identity, connection, fmt.Sprintf("payroll-%d", index), connection.Accounts[0].ID, date, -120000, func(r *budget.ActivityRecord) {
			r.Name = "Fictional payroll"
			r.Counterparty = "Fictional employer"
			r.PrimaryCategory = "INCOME"
			r.DetailedCategory = "INCOME_WAGES"
		})
	}
	projection, err := s.MonthlyActivity(t.Context(), identity, today, "household", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Suggestions) != 1 {
		t.Fatalf("expected one current payroll suggestion, got %d", len(projection.Suggestions))
	}
	before, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	return s, identity, projection.Suggestions[0], before
}

func TestActivityPaycheckServiceChangesOnlySelectedPayFields(t *testing.T) {
	s, identity, suggestion, before := servicePaycheckFixture(t)
	request := budget.ApplyPaycheckRequest{ExpectedPlanRevision: before.Revision, MemberID: "member-2"}
	got, err := s.ApplyPaycheck(t.Context(), identity, suggestion.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	expected := before
	expected.Members = append([]budget.Member(nil), before.Members...)
	expected.Revision++
	expected.Members[1].MonthlyTakeHomeCents = suggestion.MonthlyTakeHomeCents
	expected.Members[1].PayFrequency = suggestion.Frequency
	expected.Members[1].PaySchedule = &suggestion.PaySchedule
	if !reflect.DeepEqual(got, expected) {
		t.Fatal("paycheck confirmation changed unrelated plan values")
	}
	saved, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, expected) {
		t.Fatal("paycheck confirmation did not persist the exact plan")
	}
	if _, err := s.ApplyPaycheck(t.Context(), identity, suggestion.ID, request); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale confirmation: %v", err)
	}
}

func TestActivityPaycheckServiceRejectsStaleLedgerAndPlan(t *testing.T) {
	s, identity, suggestion, before := servicePaycheckFixture(t)
	request := budget.ApplyPaycheckRequest{ExpectedPlanRevision: before.Revision - 1, MemberID: identity.MemberID}
	if _, err := s.ApplyPaycheck(t.Context(), identity, suggestion.ID, request); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale plan: %v", err)
	}
	request.ExpectedPlanRevision = before.Revision
	execSQL(t, s, `UPDATE bank_activity_records SET revision=revision+1 WHERE id='payroll-0'`)
	if _, err := s.ApplyPaycheck(t.Context(), identity, suggestion.ID, request); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stale ledger suggestion: %v", err)
	}
	saved, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, before) {
		t.Fatal("rejected suggestion changed plan values")
	}
}

func TestActivityPaycheckServiceRequiresCurrentHouseholdMember(t *testing.T) {
	s, identity, suggestion, before := servicePaycheckFixture(t)
	request := budget.ApplyPaycheckRequest{ExpectedPlanRevision: before.Revision, MemberID: "missing-member"}
	if _, err := s.ApplyPaycheck(t.Context(), identity, suggestion.ID, request); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing selected member: %v", err)
	}
	identity.MemberID = "missing-member"
	request.MemberID = "member-1"
	if _, err := s.ApplyPaycheck(t.Context(), identity, suggestion.ID, request); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing session member: %v", err)
	}
	saved, err := s.db.Read(t.Context(), identity.HouseholdID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, before) {
		t.Fatal("rejected membership changed plan values")
	}
}
