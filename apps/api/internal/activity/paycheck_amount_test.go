package activity

import (
	"reflect"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestPaycheckMedianVariabilityAndMonthlyRounding(t *testing.T) {
	tests := []struct {
		name             string
		amounts          []int64
		typical, monthly int64
		variable         bool
	}{
		{"small cent changes", []int64{120000, 120001, 120002}, 120001, 520004, false},
		{"variable payroll", []int64{100000, 120000, 150000}, 120000, 520000, true},
		{"even median rounds half cent", []int64{100000, 120000, 120001, 150000}, 120001, 520004, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dates := []string{"2026-09-18", "2026-09-25", "2026-10-02"}
			if len(test.amounts) == 4 {
				dates = append([]string{"2026-09-11"}, dates...)
			}
			records, accounts := paycheckFixture(dates)
			for index, amount := range test.amounts {
				cents := -amount
				records[index].AmountCents = &cents
			}
			got := Suggestions(records, accounts, paycheckTestTime(t, "2026-10-03"))
			if len(got) != 1 {
				t.Fatalf("expected one suggestion, got %d", len(got))
			}
			if got[0].TypicalPaycheckCents != test.typical || got[0].MonthlyTakeHomeCents != test.monthly || got[0].VariableAmount != test.variable {
				t.Fatalf("wrong median, rounding or variability: %+v", got[0])
			}
		})
	}
}

func TestPaycheckBoundsMonthlyIncome(t *testing.T) {
	records, accounts := paycheckFixture([]string{"2026-09-18", "2026-09-25", "2026-10-02"})
	for index := range records {
		amount := -budget.MaxMoneyCents
		records[index].AmountCents = &amount
	}
	if got := Suggestions(records, accounts, paycheckTestTime(t, "2026-10-03")); len(got) != 0 {
		t.Fatal("monthly estimate above the plan limit produced a suggestion")
	}
	monthly, accounts := paycheckFixture([]string{"2026-08-02", "2026-09-02", "2026-10-02"})
	for index := range monthly {
		amount := -budget.MaxMoneyCents
		monthly[index].AmountCents = &amount
	}
	got := Suggestions(monthly, accounts, paycheckTestTime(t, "2026-10-03"))
	if len(got) != 1 || got[0].MonthlyTakeHomeCents != budget.MaxMoneyCents {
		t.Fatal("the exact monthly plan limit was rejected")
	}
}

func TestPaycheckFingerprintIsStableAndProtectsChangedFacts(t *testing.T) {
	records, accounts := paycheckFixture([]string{"2026-09-18", "2026-09-25", "2026-10-02"})
	now := paycheckTestTime(t, "2026-10-03")
	before := append([]budget.ActivityRecord{}, records...)
	baseline := Suggestions(records, accounts, now)
	if len(baseline) != 1 || len(baseline[0].ID) != 64 || !reflect.DeepEqual(records, before) {
		t.Fatal("suggestion fingerprint is absent or inference mutated input")
	}
	reversed := []budget.ActivityRecord{records[2], records[1], records[0]}
	if got := Suggestions(reversed, accounts, now); len(got) != 1 || got[0].ID != baseline[0].ID {
		t.Fatal("input order changed the fingerprint")
	}
	for _, mutation := range []func([]budget.ActivityRecord, map[string]budget.ActivityAccount){
		func(rows []budget.ActivityRecord, _ map[string]budget.ActivityAccount) { rows[2].Revision++ },
		func(rows []budget.ActivityRecord, _ map[string]budget.ActivityAccount) {
			amount := int64(-120001)
			rows[2].AmountCents = &amount
		},
		func(rows []budget.ActivityRecord, _ map[string]budget.ActivityAccount) { rows[2].ID = "corrected-id" },
		func(_ []budget.ActivityRecord, selected map[string]budget.ActivityAccount) {
			account := selected["checking"]
			account.Ownership = "shared"
			selected["checking"] = account
		},
	} {
		rows := append([]budget.ActivityRecord{}, records...)
		selected := map[string]budget.ActivityAccount{"checking": accounts["checking"]}
		mutation(rows, selected)
		got := Suggestions(rows, selected, now)
		if len(got) != 1 || got[0].ID == baseline[0].ID {
			t.Fatal("changed source facts retained a valid stale fingerprint")
		}
	}
}
