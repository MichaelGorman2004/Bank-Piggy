package budget

import (
	"math"
	"reflect"
	"testing"
)

func payrollPlan() Household {
	plan := Blank("First")
	plan.Members[0].AnnualBaseSalaryCents = 12000000
	plan.Members[0].MonthlyTakeHomeCents = 700000
	plan.Members[0].Accounts = []AccountBalance{{ID: "retirement", Name: "Retirement", Type: "traditional_401k", BalanceCents: 12345}}
	return plan
}

func tieredPayroll() *Payroll {
	return &Payroll{Traditional401KBps: 600, Roth401KBps: 400, MatchMethod: "tiers", MatchTiers: []MatchTier{
		{UpToContributionBps: 300, MatchRateBps: 10000}, {UpToContributionBps: 500, MatchRateBps: 5000},
	}}
}

func TestPayrollEstimatesDoNotDeductFromTakeHomeOrChangeBalances(t *testing.T) {
	plan := payrollPlan()
	before := Response(plan, testMonth(t, "2026-11"))
	plan.Members[0].Payroll = tieredPayroll()
	if err := Validate(plan); err != nil {
		t.Fatal(err)
	}
	after := Response(plan, testMonth(t, "2026-11"))
	want := PayrollSummary{MemberID: "member-1", Traditional401KMonthlyCents: 60000, Roth401KMonthlyCents: 40000, EmployerMonthlyCents: 40000}
	if len(after.PayrollSummaries) != 1 || after.PayrollSummaries[0] != want {
		t.Fatalf("unexpected payroll estimate: %+v", after.PayrollSummaries)
	}
	if !reflect.DeepEqual(before.Summary, after.Summary) || !reflect.DeepEqual(before.YearlySummary, after.YearlySummary) {
		t.Fatal("payroll estimates changed spendable cash or account snapshots")
	}
}

func TestEmployerMatchAppliesEachRateOnlyToItsContributionSlice(t *testing.T) {
	for _, test := range []struct{ contribution, employer int64 }{{0, 0}, {200, 20000}, {300, 30000}, {400, 35000}, {500, 40000}, {1200, 40000}} {
		plan := payrollPlan()
		plan.Members[0].Payroll = tieredPayroll()
		plan.Members[0].Payroll.Traditional401KBps = test.contribution
		plan.Members[0].Payroll.Roth401KBps = 0
		got := Response(plan, testMonth(t, "2026-11")).PayrollSummaries[0].EmployerMonthlyCents
		if got != test.employer {
			t.Fatalf("contribution %d: got %d employer cents, want %d", test.contribution, got, test.employer)
		}
	}
}

func TestPayrollRoundsCombinedMatchOnceAndSupportsLargeSalary(t *testing.T) {
	plan := payrollPlan()
	plan.Members[0].AnnualBaseSalaryCents = 600
	plan.Members[0].Payroll = &Payroll{Traditional401KBps: 100, Roth401KBps: 100, MatchMethod: "tiers", MatchTiers: []MatchTier{{100, 10000}, {200, 10000}}}
	summary := Response(plan, testMonth(t, "2026-11")).PayrollSummaries[0]
	if summary.EmployerMonthlyCents != 1 || summary.Traditional401KMonthlyCents != 1 || summary.Roth401KMonthlyCents != 0 {
		t.Fatalf("half-cent estimates were rounded incorrectly: %+v", summary)
	}
	plan.Members[0].AnnualBaseSalaryCents = MaxMoneyCents
	plan.Members[0].Payroll = &Payroll{Traditional401KBps: 10000, MatchMethod: "tiers", MatchTiers: []MatchTier{{10000, 20000}}}
	if err := Validate(plan); err != nil {
		t.Fatal(err)
	}
	summary = Response(plan, testMonth(t, "2026-11")).PayrollSummaries[0]
	if summary.EmployerMonthlyCents != 16666666667 || summary.Traditional401KMonthlyCents != 8333333333 {
		t.Fatalf("large payroll estimate overflowed: %+v", summary)
	}
}

func TestEmployeePayrollReconcilesCentsWithZeroAndFullRateBoundaries(t *testing.T) {
	for _, test := range []struct{ salary, traditionalBps, rothBps, traditional, roth int64 }{
		{0, 10000, 0, 0, 0}, {12, 0, 10000, 0, 1}, {12, 10000, 0, 1, 0},
		{12, 5000, 5000, 1, 0}, {600, 100, 100, 1, 0},
		{MaxMoneyCents, 5000, 5000, 4166666667, 4166666666},
	} {
		plan := payrollPlan()
		plan.Members[0].AnnualBaseSalaryCents = test.salary
		plan.Members[0].Payroll = &Payroll{Traditional401KBps: test.traditionalBps, Roth401KBps: test.rothBps, MatchMethod: "none", MatchTiers: []MatchTier{}}
		if err := Validate(plan); err != nil {
			t.Fatal(err)
		}
		summary := Response(plan, testMonth(t, "2026-11")).PayrollSummaries[0]
		if summary.Traditional401KMonthlyCents != test.traditional || summary.Roth401KMonthlyCents != test.roth {
			t.Fatalf("salary %d with %d/%d bps lost employee cents: %+v", test.salary, test.traditionalBps, test.rothBps, summary)
		}
	}
}

func TestFixedEmployerContributionAndAbsentPayrollRemainSeparate(t *testing.T) {
	plan := Blank("First", "Second")
	plan.Members[0].Payroll = &Payroll{MatchMethod: "fixed", MatchTiers: []MatchTier{}, EmployerMonthlyCents: 12345}
	if err := Validate(plan); err != nil {
		t.Fatal(err)
	}
	response := Response(plan, testMonth(t, "2026-11"))
	want := []PayrollSummary{{MemberID: "member-1", EmployerMonthlyCents: 12345}, {MemberID: "member-2"}}
	if !reflect.DeepEqual(response.PayrollSummaries, want) || response.Summary.Household != (Summary{}) {
		t.Fatalf("fixed contribution changed cash flow or missing-member summaries: %+v", response)
	}
}

func TestPayrollRejectsInvalidContributionsAndConflictingMatchMethods(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Payroll)
	}{
		{"negative traditional", func(p *Payroll) { p.Traditional401KBps = -1 }},
		{"negative roth", func(p *Payroll) { p.Roth401KBps = -1 }},
		{"combined over salary", func(p *Payroll) { p.Roth401KBps = 10000 }},
		{"overflow rate", func(p *Payroll) { p.Traditional401KBps = math.MaxInt64 }},
		{"nil tiers", func(p *Payroll) { p.MatchTiers = nil }},
		{"empty tiers", func(p *Payroll) { p.MatchTiers = []MatchTier{} }},
		{"reversed tiers", func(p *Payroll) { p.MatchTiers[1].UpToContributionBps = 200 }},
		{"zero threshold", func(p *Payroll) { p.MatchTiers[0].UpToContributionBps = 0 }},
		{"high threshold", func(p *Payroll) { p.MatchTiers[1].UpToContributionBps = 10001 }},
		{"negative match", func(p *Payroll) { p.MatchTiers[0].MatchRateBps = -1 }},
		{"high match", func(p *Payroll) { p.MatchTiers[0].MatchRateBps = 20001 }},
		{"too many tiers", func(p *Payroll) { p.MatchTiers = make([]MatchTier, 6) }},
		{"fixed conflict", func(p *Payroll) { p.MatchMethod = "fixed" }},
		{"none conflict", func(p *Payroll) { p.MatchMethod = "none" }},
		{"unknown method", func(p *Payroll) { p.MatchMethod = "other" }},
		{"tier fixed conflict", func(p *Payroll) { p.EmployerMonthlyCents = 1 }},
		{"invalid employer cents", func(p *Payroll) { p.EmployerMonthlyCents = -1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := payrollPlan()
			plan.Members[0].Payroll = tieredPayroll()
			test.change(plan.Members[0].Payroll)
			if Validate(plan) == nil {
				t.Fatal("invalid payroll was accepted")
			}
		})
	}
}
