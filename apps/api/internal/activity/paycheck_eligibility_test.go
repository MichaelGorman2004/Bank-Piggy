package activity

import (
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestPaycheckRejectsIneligibleFacts(t *testing.T) {
	tests := []struct {
		name   string
		change func(*budget.ActivityRecord)
	}{
		{"pending", func(record *budget.ActivityRecord) { record.Pending = true }},
		{"cancelled", func(record *budget.ActivityRecord) { record.Cancelled = true }},
		{"investment", func(record *budget.ActivityRecord) { record.Kind = "investment" }},
		{"not payroll", func(record *budget.ActivityRecord) { record.Category = "income" }},
		{"missing amount", func(record *budget.ActivityRecord) { record.AmountCents = nil }},
		{"zero amount", func(record *budget.ActivityRecord) { amount := int64(0); record.AmountCents = &amount }},
		{"debit", func(record *budget.ActivityRecord) { amount := int64(120000); record.AmountCents = &amount }},
		{"missing currency", func(record *budget.ActivityRecord) { record.Currency = nil }},
		{"other currency", func(record *budget.ActivityRecord) { currency := "EUR"; record.Currency = &currency }},
		{"future", func(record *budget.ActivityRecord) { record.Date = "2026-10-09" }},
		{"invalid date", func(record *budget.ActivityRecord) { record.Date = "2026-09-31" }},
		{"no payer", func(record *budget.ActivityRecord) { record.Counterparty = " "; record.Name = " " }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			records, accounts := paycheckFixture([]string{"2026-09-18", "2026-09-25", "2026-10-02"})
			test.change(&records[2])
			if got := Suggestions(records, accounts, paycheckTestTime(t, "2026-10-03")); len(got) != 0 {
				t.Fatalf("ineligible fact completed a payroll pattern: %+v", got)
			}
		})
	}
}

func TestPaycheckRequiresSelectedAvailableUSDAccount(t *testing.T) {
	tests := []struct {
		name   string
		change func(*budget.ActivityAccount)
	}{
		{"disabled", func(account *budget.ActivityAccount) { account.Enabled = false }},
		{"unavailable", func(account *budget.ActivityAccount) { account.Available = false }},
		{"missing currency", func(account *budget.ActivityAccount) { account.Currency = nil }},
		{"other currency", func(account *budget.ActivityAccount) { currency := "CAD"; account.Currency = &currency }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			records, accounts := paycheckFixture([]string{"2026-09-18", "2026-09-25", "2026-10-02"})
			account := accounts["checking"]
			test.change(&account)
			accounts["checking"] = account
			if got := Suggestions(records, accounts, paycheckTestTime(t, "2026-10-03")); len(got) != 0 {
				t.Fatalf("unselected account produced payroll: %+v", got)
			}
		})
	}
	records, _ := paycheckFixture([]string{"2026-09-18", "2026-09-25", "2026-10-02"})
	if got := Suggestions(records, map[string]budget.ActivityAccount{}, paycheckTestTime(t, "2026-10-03")); len(got) != 0 {
		t.Fatal("an account absent from the selected set produced payroll")
	}
}

func TestPaycheckIgnoresOldAndRemovedDeposits(t *testing.T) {
	records, accounts := paycheckFixture([]string{"2026-03-13", "2026-03-20", "2026-03-27"})
	if got := Suggestions(records, accounts, paycheckTestTime(t, "2026-10-03")); len(got) != 0 {
		t.Fatal("records outside the 180-day window produced payroll")
	}
	records, accounts = paycheckFixture([]string{"2026-09-18", "2026-09-25", "2026-10-02"})
	if got := Suggestions(records[:2], accounts, paycheckTestTime(t, "2026-10-03")); len(got) != 0 {
		t.Fatal("removed deposit remained in the inferred schedule")
	}
}

func TestPaycheckGroupsPayersAndAccountsWithoutGuessingSharedOwner(t *testing.T) {
	records, accounts := paycheckFixture([]string{"2026-09-18", "2026-09-25", "2026-10-02"})
	account := accounts["checking"]
	account.Ownership = "shared"
	accounts[account.AccountID] = account
	second, _ := paycheckFixture([]string{"2026-09-18", "2026-09-25", "2026-10-02"})
	for index := range second {
		second[index].ID += "other"
		second[index].Counterparty = "Other Employer"
	}
	results := Suggestions(append(records, second...), accounts, paycheckTestTime(t, "2026-10-03"))
	if len(results) != 2 || results[0].Source == results[1].Source {
		t.Fatalf("different payers were merged: %+v", results)
	}
	for _, result := range results {
		if result.MemberID != nil || result.Ownership != "shared" || result.DepositCount != 3 {
			t.Fatalf("shared account ownership identified a wage earner: %+v", result)
		}
	}
	account.AccountID, account.Name = "other-account", "Other checking"
	accounts[account.AccountID] = account
	for index := range second {
		second[index].AccountID, second[index].Counterparty = account.AccountID, records[index].Counterparty
	}
	if got := Suggestions(append(records, second...), accounts, paycheckTestTime(t, "2026-10-03")); len(got) != 2 {
		t.Fatal("same payer in different accounts was merged")
	}
}

func TestPaycheckNormalizesPayerAndFallsBackToDescription(t *testing.T) {
	records, accounts := paycheckFixture([]string{"2026-09-18", "2026-09-25", "2026-10-02"})
	records[0].Counterparty = " ACME   Payroll "
	records[1].Counterparty, records[1].Name = "", "Acme Payroll"
	got := Suggestions(records, accounts, paycheckTestTime(t, "2026-10-03"))
	if len(got) != 1 || got[0].Source != "Checking · acme payroll" {
		t.Fatalf("equivalent payer labels did not form one stream: %+v", got)
	}
}
