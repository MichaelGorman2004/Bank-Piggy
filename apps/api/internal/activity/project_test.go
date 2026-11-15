package activity

import (
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
)

var testMonth = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
var testNow = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

func sourcesFor(accounts ...string) []budget.ActivitySource {
	rows := []budget.ActivityAccount{}
	for _, id := range accounts {
		rows = append(rows, budget.ActivityAccount{AccountID: id, Name: id, Ownership: "shared", Enabled: true, Available: true})
	}
	return []budget.ActivitySource{{ConnectionID: "source", Enabled: true, Allowed: true, State: "ready", HistoryStart: pointer("2026-08-01"), HistoryEnd: pointer("2026-11-13"), Accounts: rows}}
}

func projectFor(t *testing.T, records []budget.ActivityRecord, sources []budget.ActivitySource, scope string) budget.ActivityResponse {
	t.Helper()
	r, err := Project(budget.Blank("One", "Two"), records, sources, testMonth, scope, testNow)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestMonthlyActualsUsePostedHistoryAndExactSharedCents(t *testing.T) {
	records := []budget.ActivityRecord{
		bankRecord("pay", "2026-11-01", -10001, "INCOME", "INCOME_WAGES"),
		bankRecord("food", "2026-11-02", 4001, "FOOD_AND_DRINK", ""),
		bankRecord("refund", "2026-11-03", -1001, "FOOD_AND_DRINK", ""),
		bankRecord("card", "2026-11-04", 4000, "LOAN_PAYMENTS", ""),
		bankRecord("old", "2026-10-20", 9999, "FOOD_AND_DRINK", ""),
		investmentRecord("deposit", "roth", "2026-11-02", "cash", "deposit", -1001),
		investmentRecord("withdraw", "roth", "2026-11-03", "cash", "withdrawal", 101),
		investmentRecord("buy", "roth", "2026-11-04", "buy", "buy", 400),
		investmentRecord("dividend", "roth", "2026-11-05", "cash", "dividend", -51),
		investmentRecord("reinvest", "roth", "2026-11-05", "buy", "dividend reinvestment", 51),
		investmentRecord("fee", "roth", "2026-11-06", "fee", "fee", 11),
	}
	sources := sourcesFor("checking", "roth")
	household := projectFor(t, records, sources, "household")
	first := projectFor(t, records, sources, "member-1")
	second := projectFor(t, records, sources, "member-2")
	want := budget.ActivityTotals{IncomeCents: 10001, SpendingCents: 3000, InvestmentDepositsCents: 1001, InvestmentWithdrawalsCents: 101, NetInvestmentContributionsCents: 900, InvestmentIncomeCents: 51, InvestmentFeesCents: 11}
	if household.Totals != want {
		t.Fatalf("unexpected actuals: %+v", household.Totals)
	}
	if first.Totals.IncomeCents+second.Totals.IncomeCents != want.IncomeCents || first.Totals.NetInvestmentContributionsCents+second.Totals.NetInvestmentContributionsCents != want.NetInvestmentContributionsCents {
		t.Fatal("shared cents do not reconcile")
	}
}

func TestCoverageDoesNotRepresentMissingOrUnreviewedHistoryAsComplete(t *testing.T) {
	sources := sourcesFor("checking")
	if projectFor(t, nil, sources, "household").Coverage != "ready" {
		t.Fatal("complete empty history is not ready")
	}
	sources[0].State = "initializing"
	if projectFor(t, nil, sources, "household").Coverage != "incomplete" {
		t.Fatal("initial history displayed as complete")
	}
	sources[0].State = "ready"
	sources[0].HistoryStart = pointer("2026-11-02")
	if projectFor(t, nil, sources, "household").Coverage != "incomplete" {
		t.Fatal("missing month start displayed as complete")
	}
	sources[0].HistoryStart = pointer("2026-08-01")
	unknown := bankRecord("unknown", "2026-11-05", 100, "OTHER", "")
	result := projectFor(t, []budget.ActivityRecord{unknown}, sources, "household")
	if result.Coverage != "incomplete" || result.Totals.SpendingCents != 0 {
		t.Fatal("unreviewed record entered complete totals")
	}
	sources[0].Allowed = false
	if projectFor(t, recordsWithKnownSpending(), sources, "household").Totals.SpendingCents != 0 {
		t.Fatal("closed product gate exposed history actuals")
	}
}

func TestTotalsRejectUnsafeSums(t *testing.T) {
	records := []budget.ActivityRecord{bankRecord("a", "2026-11-01", maxSafeCents, "FOOD_AND_DRINK", ""), bankRecord("b", "2026-11-02", 1, "FOOD_AND_DRINK", "")}
	_, err := Project(budget.Blank("One"), records, sourcesFor("checking"), testMonth, "household", testNow)
	if err != ErrInvalid {
		t.Fatal("unsafe numeric total was not rejected")
	}
}

func TestSelectedUnavailableAccountKeepsCoverageIncomplete(t *testing.T) {
	sources := sourcesFor("checking", "lost-investment")
	sources[0].Accounts[1].Available = false
	if projectFor(t, nil, sources, "household").Coverage != "incomplete" {
		t.Fatal("lost selected account was treated as complete")
	}
	sources[0].Accounts[0].Enabled = false
	if projectFor(t, nil, sources, "household").Coverage != "incomplete" {
		t.Fatal("lost selected account was treated as disabled")
	}
	sources[0].Accounts[1].Enabled = false
	if projectFor(t, nil, sources, "household").Coverage != "disabled" {
		t.Fatal("explicit source exclusion was not preserved")
	}
}

func TestUnsupportedPostedCurrencyMakesZeroIncomplete(t *testing.T) {
	r := bankRecord("foreign", "2026-11-01", 1000, "FOOD_AND_DRINK", "")
	r.Currency = pointer("EUR")
	result := projectFor(t, []budget.ActivityRecord{r}, sourcesFor("checking"), "household")
	if result.Coverage != "incomplete" || result.Totals.SpendingCents != 0 {
		t.Fatal("unsupported amount became a complete zero")
	}
}

func TestFuturePostingDatesDoNotEnterCurrentActuals(t *testing.T) {
	records := []budget.ActivityRecord{bankRecord("future", "2026-11-14", 1000, "FOOD_AND_DRINK", ""), investmentRecord("future-invested", "roth", "2026-11-14", "buy", "contribution", 10000)}
	result := projectFor(t, records, sourcesFor("checking", "roth"), "household")
	if result.Totals.SpendingCents != 0 || result.Totals.NetInvestmentContributionsCents != 0 {
		t.Fatal("future activity entered actuals or disappeared")
	}
	monthly, _, err := monthlyRecords(Reconcile(records), IncludedAccounts(sourcesFor("checking", "roth")), budget.Blank("One", "Two"), "2026-11", "household", testNow)
	if err != nil || len(monthly) != 2 {
		t.Fatal("future records were lost from internal classification", err)
	}
	for _, record := range monthly {
		if record.Category != "excluded" || record.ReviewReason == nil {
			t.Fatal("future-date exclusion is not explained")
		}
	}
}

func recordsWithKnownSpending() []budget.ActivityRecord {
	return []budget.ActivityRecord{bankRecord("known", "2026-11-05", 100, "FOOD_AND_DRINK", "")}
}
