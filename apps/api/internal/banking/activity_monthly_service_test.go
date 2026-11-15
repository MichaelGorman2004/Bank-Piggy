package banking

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
)

func serviceActivityFixture(t *testing.T, investment bool) (*Service, Identity, budget.Connection) {
	t.Helper()
	s, _, identity, connection := activityTestService(t)
	if investment {
		execSQL(t, s, `UPDATE bank_connections SET kind='investment' WHERE id=$1`, connection.ID)
		execSQL(t, s, `UPDATE bank_accounts SET snapshot=jsonb_set(snapshot,'{providerType}','"investment"') WHERE connection_id=$1`, connection.ID)
		connection.Kind = "investment"
	}
	enableActivityForTest(t, s, identity, connection)
	return s, identity, connection
}

func saveServiceActivity(t *testing.T, s *Service, identity Identity, connection budget.Connection, id, account, date string, amount int64, modify func(*budget.ActivityRecord)) {
	t.Helper()
	record := budget.ActivityRecord{Kind: "bank", Date: budget.CalendarDate(date), Name: "Fictional purchase", Currency: ptr("USD"), AmountCents: ptr(amount), PrimaryCategory: "FOOD_AND_DRINK", CategoryConfidence: "VERY_HIGH"}
	if connection.Kind == "investment" {
		record.Kind = "investment"
	}
	if modify != nil {
		modify(&record)
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	execSQL(t, s, `INSERT INTO bank_activity_records(id,household_id,connection_id,account_id,provider_id,kind,posted_on,data) VALUES($1,$2,$3,$4,$1,$5,$6,$7)`, id, identity.HouseholdID, connection.ID, account, record.Kind, date, data)
}

func serviceActivityMonth(t *testing.T, s *Service, identity Identity, scope string) budget.ActivityResponse {
	t.Helper()
	result, err := s.MonthlyActivity(t.Context(), identity, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), scope)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func addServiceInvestmentAccount(t *testing.T, s *Service, identity Identity, connection budget.Connection) string {
	t.Helper()
	id := "second-investment-account"
	execSQL(t, s, `INSERT INTO bank_accounts(id,household_id,connection_id,provider_id,snapshot) SELECT $1,household_id,connection_id,'second-provider-account',snapshot FROM bank_accounts WHERE id=$2`, id, connection.Accounts[0].ID)
	selections := []budget.ActivityAccountSelection{{AccountID: connection.Accounts[0].ID, Enabled: true, Ownership: identity.MemberID}, {AccountID: id, Enabled: true, Ownership: identity.MemberID}}
	data, err := json.Marshal(selections)
	if err != nil {
		t.Fatal(err)
	}
	execSQL(t, s, `UPDATE bank_activity_jobs SET accounts=$2 WHERE connection_id=$1`, connection.ID, data)
	return id
}

func TestActivityMonthlyServiceTotalsIncludeFullSavedLedger(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, false)
	for index := 0; index < 201; index++ {
		saveServiceActivity(t, s, identity, connection, fmt.Sprintf("purchase-%03d", index), connection.Accounts[0].ID, "2026-11-03", 101, nil)
	}
	result := serviceActivityMonth(t, s, identity, "household")
	if result.Totals.SpendingCents != 20301 || result.Comparison.ExpenseRemainingCents != result.Comparison.ExpenseBudgetCents-20301 {
		t.Fatal("aggregate response omitted saved spending")
	}
}

func TestActivityMonthlyServiceReconcilesSharedCents(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, false)
	execSQL(t, s, `UPDATE bank_activity_jobs SET accounts=jsonb_set(accounts,'{0,ownership}','"shared"') WHERE connection_id=$1`, connection.ID)
	saveServiceActivity(t, s, identity, connection, "shared-purchase", connection.Accounts[0].ID, "2026-11-03", 101, nil)
	household := serviceActivityMonth(t, s, identity, "household")
	first := serviceActivityMonth(t, s, identity, "member-1")
	second := serviceActivityMonth(t, s, identity, "member-2")
	if household.Totals.SpendingCents != 101 || first.Totals.SpendingCents != 51 || second.Totals.SpendingCents != 50 {
		t.Fatal("shared cents were duplicated or lost")
	}
	execSQL(t, s, `UPDATE bank_activity_jobs SET accounts=jsonb_set(accounts,'{0,ownership}','"member-1"') WHERE connection_id=$1`, connection.ID)
	if got := serviceActivityMonth(t, s, identity, "member-2"); got.Totals.SpendingCents != 0 {
		t.Fatal("personal scope included another member's activity")
	}
}

func TestActivityMonthlyServiceFindsCrossMonthTransfer(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, true)
	second := addServiceInvestmentAccount(t, s, identity, connection)
	saveServiceActivity(t, s, identity, connection, "withdrawal", connection.Accounts[0].ID, "2026-10-31", 10001, func(r *budget.ActivityRecord) { r.ActivitySubtype = "withdrawal" })
	saveServiceActivity(t, s, identity, connection, "deposit", second, "2026-11-01", -10001, func(r *budget.ActivityRecord) { r.ActivitySubtype = "contribution" })
	result := serviceActivityMonth(t, s, identity, "household")
	if result.Totals.NetInvestmentContributionsCents != 0 {
		t.Fatal("unconfirmed cross-month transfer entered investment totals")
	}
}
