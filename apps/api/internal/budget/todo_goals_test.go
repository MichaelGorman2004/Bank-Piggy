package budget

import (
	"reflect"
	"testing"
)

func TestGoalChecklistReconcilesMonthlyCentsAcrossActualPaydays(t *testing.T) {
	tests := []struct {
		name, frequency string
		anchor          CalendarDate
		days            []DayOfMonth
		paydays         []string
		amounts         []int64
	}{
		{"four weekly", "weekly", "2026-10-30", nil, []string{"2026-11-06", "2026-11-13", "2026-11-20", "2026-11-27"}, []int64{26, 25, 25, 25}},
		{"five weekly", "weekly", "2026-07-03", nil, []string{"2026-07-03", "2026-07-10", "2026-07-17", "2026-07-24", "2026-07-31"}, []int64{21, 20, 20, 20, 20}},
		{"three biweekly", "biweekly", "2026-11-01", nil, []string{"2026-11-01", "2026-11-15", "2026-11-29"}, []int64{34, 34, 33}},
		{"semimonthly", "semimonthly", "2026-11-15", []DayOfMonth{15, 31}, []string{"2026-11-15", "2026-11-30"}, []int64{51, 50}},
		{"monthly", "monthly", "2026-01-31", nil, []string{"2026-11-30"}, []int64{101}},
		{"anchor midmonth", "weekly", "2026-11-20", nil, []string{"2026-11-20", "2026-11-27"}, []int64{51, 50}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := todoPlan(test.frequency, test.anchor, test.days...)
			var total int64
			for index, date := range test.paydays {
				result := Todos(plan, testDate(t, date))
				if len(result.Tasks) != 1 || result.Tasks[0].AmountCents != test.amounts[index] {
					t.Fatalf("payday %s did not preserve monthly rounding: %+v", date, result.Tasks)
				}
				total += result.Tasks[0].AmountCents
			}
			if total != plan.Goals[0].MonthlyAmountCents {
				t.Fatalf("monthly goal cents were lost or duplicated: %d", total)
			}
		})
	}
}

func TestSharedGoalKeepsEachMembersShareWithDifferentSchedules(t *testing.T) {
	plan := todoPlan("weekly", "2026-10-30")
	second := Blank("First", "Second").Members[1]
	second.PayFrequency = "biweekly"
	second.PaySchedule = &PaySchedule{AnchorDate: "2026-11-06", SemiMonthlyDays: []DayOfMonth{}}
	plan.Members = append(plan.Members, second)
	plan.Goals[0].OwnerID = "shared"
	monthly := map[string]int64{}
	for _, date := range []string{"2026-11-06", "2026-11-13", "2026-11-20", "2026-11-27"} {
		for _, task := range Todos(plan, testDate(t, date)).Tasks {
			if string(task.PeriodStart) == date {
				monthly[task.MemberID] += task.AmountCents
			}
		}
	}
	if monthly["member-1"] != 51 || monthly["member-2"] != 50 {
		t.Fatalf("shared goal cents did not reconcile to monthly shares: %+v", monthly)
	}
}

func TestGoalChecklistIsStableAndDoesNotChangeSavedProgress(t *testing.T) {
	plan := todoPlan("biweekly", "2026-11-06")
	plan.Revision = 7
	plan.Goals[0].SavingsTarget = &SavingsTarget{TargetAmountCents: 10000, SavedAmountCents: 1234}
	before := Response(plan, testMonth(t, "2026-11"))
	first := Todos(plan, testDate(t, "2026-11-06"))
	second := Todos(plan, testDate(t, "2026-11-19"))
	if !reflect.DeepEqual(first.Tasks, second.Tasks) || first.PlanRevision != 7 {
		t.Fatal("task identities or amounts changed within one pay period")
	}
	task := first.Tasks[0]
	if task.ID != "member-1:goal:goal:2026-11-06" || task.DueDate != "2026-11-19" || task.Completed {
		t.Fatalf("unexpected task identity or default completion: %+v", task)
	}
	first.Tasks[0].Completed = true
	after := Response(plan, testMonth(t, "2026-11"))
	if !reflect.DeepEqual(before, after) || plan.Goals[0].SavingsTarget.SavedAmountCents != 1234 {
		t.Fatal("checklist generation or acknowledgement changed plan money")
	}
}

func TestContributionYearUsesPayPeriodStartYear(t *testing.T) {
	plan := todoPlan("biweekly", "2026-12-25")
	year := ContributionYear(2026)
	plan.Goals[0].ContributionYear = &year
	plan.Goals[0].SavingsTarget = &SavingsTarget{TargetAmountCents: 700000, SavedAmountCents: 500000}
	if err := Validate(plan); err != nil {
		t.Fatal(err)
	}
	if tasks := Todos(plan, testDate(t, "2027-01-01")).Tasks; len(tasks) != 1 || tasks[0].PeriodStart != "2026-12-25" {
		t.Fatal("a December contribution task was removed before its pay period ended", tasks)
	}
	if tasks := Todos(plan, testDate(t, "2027-01-08")).Tasks; len(tasks) != 0 {
		t.Fatal("a previous-year goal produced a new-year task", tasks)
	}
	plan.Goals[0].MonthlyAmountCents = 0
	if err := Validate(plan); err != nil {
		t.Fatal("zero allocation for annual progress was rejected", err)
	}
	if tasks := Todos(plan, testDate(t, "2026-12-25")).Tasks; len(tasks) != 0 {
		t.Fatal("zero monthly allocation created a task", tasks)
	}
}

func TestContributionYearRequiresPersonalOrdinaryProgress(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Goal)
	}{
		{"shared owner", func(g *Goal) { g.OwnerID = "shared" }},
		{"missing target", func(g *Goal) { g.SavingsTarget = nil }},
		{"trip", func(g *Goal) { g.SavingsTarget = nil; g.Trip = tripGoal().Trip }},
		{"purchase", func(g *Goal) { g.SavingsTarget = nil; g.Purchase = purchaseGoal().Purchase }},
		{"early year", func(g *Goal) { year := ContributionYear(1999); g.ContributionYear = &year }},
		{"late year", func(g *Goal) { year := ContributionYear(2101); g.ContributionYear = &year }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := todoPlan("monthly", "2026-01-01")
			year := ContributionYear(2026)
			plan.Goals[0].ContributionYear = &year
			plan.Goals[0].SavingsTarget = &SavingsTarget{TargetAmountCents: 700000}
			test.change(&plan.Goals[0])
			if Validate(plan) == nil {
				t.Fatal("invalid contribution year goal was accepted")
			}
		})
	}
}

func TestContributionYearReminderStopsAtTheEnteredRemainingTarget(t *testing.T) {
	plan := todoPlan("monthly", "2026-01-01")
	year := ContributionYear(2026)
	plan.Goals[0].ContributionYear = &year
	plan.Goals[0].SavingsTarget = &SavingsTarget{TargetAmountCents: 10000, SavedAmountCents: 9950}
	before := Response(plan, testMonth(t, "2026-11"))
	tasks := Todos(plan, testDate(t, "2026-11-11")).Tasks
	if len(tasks) != 1 || tasks[0].AmountCents != 50 {
		t.Fatal("annual reminder exceeded the entered remaining contribution target", tasks)
	}
	for _, saved := range []int64{10000, 10001} {
		plan.Goals[0].SavingsTarget.SavedAmountCents = saved
		if tasks := Todos(plan, testDate(t, "2026-11-11")).Tasks; len(tasks) != 0 {
			t.Fatal("funded annual goal still produced a task", tasks)
		}
	}
	after := Response(plan, testMonth(t, "2026-11"))
	if !reflect.DeepEqual(before.Summary, after.Summary) || !reflect.DeepEqual(before.YearlySummary, after.YearlySummary) {
		t.Fatal("annual progress changed the chosen cash allocation")
	}
	plan.Goals[0].ContributionYear = nil
	if tasks := Todos(plan, testDate(t, "2026-11-11")).Tasks; len(tasks) != 1 || tasks[0].AmountCents != 101 {
		t.Fatal("ordinary goal lost its chosen contribution after reaching its target", tasks)
	}
}
