package budget

import (
	"testing"
	"time"
)

func testDate(t *testing.T, value string) time.Time {
	t.Helper()
	date, err := time.Parse(dateLayout, value)
	if err != nil {
		t.Fatal(err)
	}
	return date
}

func todoPlan(frequency string, anchor CalendarDate, days ...DayOfMonth) Household {
	plan := Blank("First")
	plan.Members[0].PayFrequency = frequency
	plan.Members[0].PaySchedule = &PaySchedule{AnchorDate: anchor, SemiMonthlyDays: append([]DayOfMonth{}, days...)}
	plan.Goals = []Goal{{ID: "goal", Name: "Save", OwnerID: "member-1", MonthlyAmountCents: 101}}
	return plan
}

func TestCurrentPayPeriodUsesActualCalendarDates(t *testing.T) {
	tests := []struct {
		name, frequency, anchor, date, start, next, due string
		days                                            []DayOfMonth
	}{
		{"weekly day before", "weekly", "2026-10-30", "2026-11-05", "2026-10-30", "2026-11-06", "2026-11-05", nil},
		{"weekly payday", "weekly", "2026-10-30", "2026-11-06", "2026-11-06", "2026-11-13", "2026-11-12", nil},
		{"biweekly year boundary", "biweekly", "2026-12-18", "2027-01-01", "2027-01-01", "2027-01-15", "2027-01-14", nil},
		{"biweekly leap day", "biweekly", "2024-02-01", "2024-02-29", "2024-02-29", "2024-03-14", "2024-03-13", nil},
		{"monthly before leap payday", "monthly", "2024-01-31", "2024-02-28", "2024-01-31", "2024-02-29", "2024-02-28", nil},
		{"monthly leap payday", "monthly", "2024-01-31", "2024-02-29", "2024-02-29", "2024-03-31", "2024-03-30", nil},
		{"monthly no February drift", "monthly", "2024-01-31", "2024-03-30", "2024-02-29", "2024-03-31", "2024-03-30", nil},
		{"monthly common February", "monthly", "2026-01-31", "2026-02-28", "2026-02-28", "2026-03-31", "2026-03-30", nil},
		{"semimonthly leap clamp", "semimonthly", "2024-01-15", "2024-02-28", "2024-02-15", "2024-02-29", "2024-02-28", []DayOfMonth{15, 31}},
		{"semimonthly leap payday", "semimonthly", "2024-01-15", "2024-02-29", "2024-02-29", "2024-03-15", "2024-03-14", []DayOfMonth{15, 31}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := todoPlan(test.frequency, CalendarDate(test.anchor), test.days...)
			if err := Validate(plan); err != nil {
				t.Fatal(err)
			}
			result := Todos(plan, testDate(t, test.date))
			if len(result.Periods) != 1 {
				t.Fatalf("expected one period: %+v", result)
			}
			period := result.Periods[0]
			if string(period.StartDate) != test.start || string(period.NextPayDate) != test.next || string(period.DueDate) != test.due {
				t.Fatalf("incorrect pay period: %+v", period)
			}
		})
	}
}

func TestChecklistSkipsMissingInvalidFutureAndOutOfRangeSchedules(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Household)
		date   string
	}{
		{"no schedule", func(h *Household) { h.Members[0].PaySchedule = nil }, "2026-11-11"},
		{"future anchor", func(h *Household) { h.Members[0].PaySchedule.AnchorDate = "2026-12-01" }, "2026-11-11"},
		{"invalid anchor", func(h *Household) { h.Members[0].PaySchedule.AnchorDate = "2026-02-30" }, "2026-11-11"},
		{"irregular", func(h *Household) { h.Members[0].PayFrequency = "irregular" }, "2026-11-11"},
		{"unsupported frequency", func(h *Household) { h.Members[0].PayFrequency = "daily" }, "2026-11-11"},
		{"last supported month", func(h *Household) { h.Members[0].PaySchedule.AnchorDate = "2100-12-01" }, "2100-12-01"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := todoPlan("monthly", "2026-01-01")
			test.change(&plan)
			result := Todos(plan, testDate(t, test.date))
			if len(result.Periods) != 0 || len(result.Tasks) != 0 || result.Periods == nil || result.Tasks == nil || result.UnscheduledExpenseIds == nil {
				t.Fatalf("unschedulable plan produced tasks or null arrays: %+v", result)
			}
		})
	}
}

func TestValidDateRequiresExactBoundedCalendarDate(t *testing.T) {
	for _, test := range []struct {
		date  string
		valid bool
	}{{"2000-02-29", true}, {"2100-12-31", true}, {"2100-02-29", false}, {"2026-02-29", false}, {"1999-12-31", false}, {"2101-01-01", false}, {"2026-2-01", false}, {"2026-01-01T00:00:00Z", false}} {
		if got := ValidDate(test.date); got != test.valid {
			t.Errorf("ValidDate(%q) = %v, want %v", test.date, got, test.valid)
		}
	}
}

func TestPayScheduleValidationRejectsAmbiguousOrInvalidDates(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Member)
	}{
		{"invalid date", func(m *Member) { m.PaySchedule.AnchorDate = "2026-02-30" }},
		{"early date", func(m *Member) { m.PaySchedule.AnchorDate = "1999-01-15" }},
		{"late date", func(m *Member) { m.PaySchedule.AnchorDate = "2101-01-15" }},
		{"missing days array", func(m *Member) { m.PaySchedule.SemiMonthlyDays = nil }},
		{"missing days", func(m *Member) { m.PaySchedule.SemiMonthlyDays = []DayOfMonth{} }},
		{"one day", func(m *Member) { m.PaySchedule.SemiMonthlyDays = []DayOfMonth{15} }},
		{"duplicate days", func(m *Member) { m.PaySchedule.SemiMonthlyDays = []DayOfMonth{15, 15} }},
		{"reversed days", func(m *Member) { m.PaySchedule.SemiMonthlyDays = []DayOfMonth{31, 15} }},
		{"February collision", func(m *Member) { m.PaySchedule.SemiMonthlyDays = []DayOfMonth{28, 31} }},
		{"zero day", func(m *Member) { m.PaySchedule.SemiMonthlyDays = []DayOfMonth{0, 15} }},
		{"large day", func(m *Member) { m.PaySchedule.SemiMonthlyDays = []DayOfMonth{15, 32} }},
		{"wrong anchor", func(m *Member) { m.PaySchedule.AnchorDate = "2026-01-16" }},
		{"monthly with days", func(m *Member) { m.PayFrequency = "monthly" }},
		{"irregular with schedule", func(m *Member) { m.PayFrequency = "irregular" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := todoPlan("semimonthly", "2026-01-15", 15, 31)
			test.change(&plan.Members[0])
			if Validate(plan) == nil {
				t.Fatal("invalid pay schedule accepted")
			}
		})
	}
}

func TestSemimonthlyScheduleAcceptsClampedAnchorAndLegacyNoSchedule(t *testing.T) {
	plan := todoPlan("semimonthly", "2026-02-28", 15, 31)
	if err := Validate(plan); err != nil {
		t.Fatal("clamped payday was rejected", err)
	}
	plan.Members[0].PaySchedule = nil
	plan.Members[0].PayFrequency = "irregular"
	if err := Validate(plan); err != nil {
		t.Fatal("legacy irregular plan without a schedule was rejected", err)
	}
}

func TestMonthlyPayUsesExplicitMonthEndInsteadOfGuessingFromFebruary(t *testing.T) {
	for _, test := range []struct {
		monthEnd bool
		next     CalendarDate
	}{{false, "2026-03-28"}, {true, "2026-03-31"}} {
		plan := todoPlan("monthly", "2026-02-28")
		plan.Members[0].PaySchedule.MonthEnd = test.monthEnd
		if err := Validate(plan); err != nil {
			t.Fatal(err)
		}
		result := Todos(plan, testDate(t, "2026-03-01"))
		if len(result.Periods) != 1 || result.Periods[0].NextPayDate != test.next {
			t.Fatalf("monthEnd=%v: unexpected period %+v", test.monthEnd, result.Periods)
		}
	}
}

func TestMonthEndPayRejectsNonmonthlyOrIncorrectAnchor(t *testing.T) {
	for _, test := range []struct {
		frequency string
		anchor    CalendarDate
	}{{"weekly", "2026-02-28"}, {"biweekly", "2026-02-28"}, {"semimonthly", "2026-02-28"}, {"monthly", "2026-02-27"}} {
		plan := todoPlan(test.frequency, test.anchor)
		if test.frequency == "semimonthly" {
			plan.Members[0].PaySchedule.SemiMonthlyDays = []DayOfMonth{15, 31}
		}
		plan.Members[0].PaySchedule.MonthEnd = true
		if Validate(plan) == nil {
			t.Fatalf("invalid month-end schedule accepted: %+v", test)
		}
	}
}
