package activity

import (
	"reflect"
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
)

type paycheckScheduleCase struct {
	name, frequency, now, next, anchor string
	dates                              []string
	days                               []budget.DayOfMonth
	monthEnd                           bool
	monthly                            int64
}

func TestPaycheckRegularSchedules(t *testing.T) {
	tests := []paycheckScheduleCase{
		{name: "weekly", frequency: "weekly", now: "2026-10-03", next: "2026-10-09", dates: []string{"2026-09-18", "2026-09-25", "2026-10-02"}, days: []budget.DayOfMonth{}, monthly: 520000},
		{name: "biweekly", frequency: "biweekly", now: "2026-10-18", next: "2026-10-30", dates: []string{"2026-09-04", "2026-09-18", "2026-10-02", "2026-10-16"}, days: []budget.DayOfMonth{}, monthly: 260000},
		{name: "semimonthly month end", frequency: "semimonthly", now: "2026-11-01", next: "2026-11-15", dates: []string{"2026-09-15", "2026-09-30", "2026-10-15", "2026-10-31"}, days: []budget.DayOfMonth{15, 31}, monthly: 240000},
		{name: "semimonthly fixed", frequency: "semimonthly", now: "2026-11-01", next: "2026-11-10", dates: []string{"2026-09-10", "2026-09-25", "2026-10-10", "2026-10-25"}, days: []budget.DayOfMonth{10, 25}, monthly: 240000},
		{name: "monthly fixed", frequency: "monthly", now: "2026-10-11", next: "2026-11-10", dates: []string{"2026-08-10", "2026-09-10", "2026-10-10"}, days: []budget.DayOfMonth{}, monthly: 120000},
		{name: "monthly month end", frequency: "monthly", now: "2026-11-01", next: "2026-11-30", dates: []string{"2026-08-31", "2026-09-30", "2026-10-31"}, days: []budget.DayOfMonth{}, monthEnd: true, monthly: 120000},
		{name: "monthly clamped February keeps original day", frequency: "monthly", now: "2027-03-01", next: "2027-03-30", anchor: "2027-01-30", dates: []string{"2026-12-30", "2027-01-30", "2027-02-28"}, days: []budget.DayOfMonth{}, monthly: 120000},
		{name: "semimonthly partial boundary months", frequency: "semimonthly", now: "2026-12-16", next: "2026-12-31", dates: []string{"2026-10-31", "2026-11-15", "2026-11-30", "2026-12-15"}, days: []budget.DayOfMonth{15, 31}, monthly: 240000},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			records, accounts := paycheckFixture(test.dates)
			results := Suggestions(records, accounts, paycheckTestTime(t, test.now))
			if len(results) != 1 {
				t.Fatalf("expected one suggestion, got %d", len(results))
			}
			assertPaycheckSchedule(t, results[0], test)
		})
	}
}

func assertPaycheckSchedule(t *testing.T, got budget.PaycheckSuggestion, expected paycheckScheduleCase) {
	t.Helper()
	if got.Frequency != expected.frequency || got.MonthlyTakeHomeCents != expected.monthly || got.TypicalPaycheckCents != 120000 {
		t.Fatalf("wrong schedule or net amount: %+v", got)
	}
	assertPaycheckDates(t, got, expected)
	assertPaycheckOwnerAndAnchor(t, got, expected)
}

func assertPaycheckDates(t *testing.T, got budget.PaycheckSuggestion, expected paycheckScheduleCase) {
	t.Helper()
	if got.NextPayDate == nil || string(*got.NextPayDate) != expected.next || string(got.LastPayDate) != expected.dates[len(expected.dates)-1] {
		t.Fatalf("wrong paycheck dates: %+v", got)
	}
	if !reflect.DeepEqual(got.PaySchedule.SemiMonthlyDays, expected.days) || got.PaySchedule.MonthEnd != expected.monthEnd {
		t.Fatalf("noncanonical pay schedule: %+v", got.PaySchedule)
	}
}

func assertPaycheckOwnerAndAnchor(t *testing.T, got budget.PaycheckSuggestion, expected paycheckScheduleCase) {
	t.Helper()
	anchor := expected.anchor
	if anchor == "" {
		anchor = string(got.LastPayDate)
	}
	if string(got.PaySchedule.AnchorDate) != anchor || got.MemberID == nil || *got.MemberID != "member-1" || got.VariableAmount {
		t.Fatalf("wrong anchor, owner or variability: %+v", got)
	}
}

func TestPaycheckRejectsUnprovedOrStaleSchedules(t *testing.T) {
	tests := []struct {
		name, now string
		dates     []string
	}{
		{"one deposit", "2026-10-03", []string{"2026-10-02"}},
		{"two weekly deposits", "2026-10-03", []string{"2026-09-25", "2026-10-02"}},
		{"three biweekly deposits", "2026-10-03", []string{"2026-09-04", "2026-09-18", "2026-10-02"}},
		{"two monthly deposits", "2026-10-03", []string{"2026-09-02", "2026-10-02"}},
		{"missing weekly payday", "2026-10-03", []string{"2026-09-04", "2026-09-11", "2026-10-02"}},
		{"missing month", "2026-10-03", []string{"2026-07-02", "2026-09-02", "2026-10-02"}},
		{"inconsistent monthly day", "2026-10-03", []string{"2026-08-01", "2026-09-02", "2026-10-03"}},
		{"stale weekly stream", "2026-10-13", []string{"2026-09-18", "2026-09-25", "2026-10-02"}},
		{"ambiguous February biweekly or semimonthly", "2027-03-16", []string{"2027-02-01", "2027-02-15", "2027-03-01", "2027-03-15"}},
		{"duplicate payday", "2026-10-03", []string{"2026-09-18", "2026-09-25", "2026-10-02", "2026-10-02"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			records, accounts := paycheckFixture(test.dates)
			if got := Suggestions(records, accounts, paycheckTestTime(t, test.now)); len(got) != 0 {
				t.Fatalf("unproved schedule produced a suggestion: %+v", got)
			}
		})
	}
}

func TestPaycheckAllowsBoundedPostingDelay(t *testing.T) {
	records, accounts := paycheckFixture([]string{"2026-09-18", "2026-09-25", "2026-10-02"})
	for _, day := range []string{"2026-10-09", "2026-10-12"} {
		if got := Suggestions(records, accounts, paycheckTestTime(t, day)); len(got) != 1 {
			t.Fatalf("expected a suggestion within three-day posting grace on %s", day)
		}
	}
	if got := Suggestions(records, accounts, paycheckTestTime(t, "2026-10-13")); len(got) != 0 {
		t.Fatal("posting grace exceeded three calendar days")
	}
}

func paycheckTestTime(t *testing.T, value string) time.Time {
	t.Helper()
	date, err := time.Parse(paycheckDateLayout, value)
	if err != nil {
		t.Fatal(err)
	}
	return date
}

func paycheckFixture(dates []string) ([]budget.ActivityRecord, map[string]budget.ActivityAccount) {
	usd := "USD"
	account := budget.ActivityAccount{AccountID: "checking", Name: "Checking", ProviderType: "depository", Currency: &usd,
		Available: true, Enabled: true, Ownership: "member-1"}
	records := make([]budget.ActivityRecord, 0, len(dates))
	for _, date := range dates {
		amount := int64(-120000)
		records = append(records, budget.ActivityRecord{ID: date, ConnectionID: "bank", AccountID: account.AccountID,
			AccountName: account.Name, Ownership: account.Ownership, Kind: "bank", Date: budget.CalendarDate(date),
			Name: "Payroll", Counterparty: "Acme Payroll", Currency: &usd, AmountCents: &amount, Category: "paycheck", Revision: 1})
	}
	return records, map[string]budget.ActivityAccount{account.AccountID: account}
}
