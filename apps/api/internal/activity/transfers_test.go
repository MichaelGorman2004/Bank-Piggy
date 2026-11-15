package activity

import (
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestPotentialTransferIsReconciledBeforeMonthAndScopeFiltering(t *testing.T) {
	withdraw := investmentRecord("out", "brokerage", "2026-10-31", "cash", "withdrawal", 50000)
	deposit := investmentRecord("in", "roth", "2026-11-02", "buy", "contribution", 50000)
	sources := sourcesFor("brokerage", "roth")
	sources[0].Accounts[0].Ownership = "member-1"
	sources[0].Accounts[1].Ownership = "member-2"
	result := projectFor(t, []budget.ActivityRecord{withdraw, deposit}, sources, "member-2")
	if len(result.Records) != 1 || result.ReviewCount != 1 || result.Totals.NetInvestmentContributionsCents != 0 || len(result.Records[0].TransferCandidates) != 1 {
		t.Fatal("cross-month potential rollover counted as new funding")
	}
	deposit.IsReviewed, deposit.Category = true, "contribution"
	result = projectFor(t, []budget.ActivityRecord{withdraw, deposit}, sources, "member-2")
	if result.Totals.NetInvestmentContributionsCents != 50000 {
		t.Fatal("explicit funding review was not preserved")
	}
	deposit.IsReviewed, deposit.Category, deposit.TransferPeerID = true, "transfer", &withdraw.ID
	withdraw.IsReviewed, withdraw.Category, withdraw.TransferPeerID = true, "transfer", &deposit.ID
	result = projectFor(t, []budget.ActivityRecord{withdraw, deposit}, sources, "member-2")
	if result.Totals.NetInvestmentContributionsCents != 0 || len(result.Records[0].TransferCandidates) != 1 {
		t.Fatal("saved peer outside current month or scope disappeared")
	}
}

func TestTransferPairRequiresOppositeFundingAndBoundedDates(t *testing.T) {
	a := investmentRecord("out", "brokerage", "2026-11-01", "cash", "withdrawal", 10000)
	b := investmentRecord("in", "roth", "2026-11-08", "buy", "contribution", 10000)
	if !ValidTransferPair(a, b) {
		t.Fatal("valid opposite funding legs were rejected")
	}
	cases := []func(*budget.ActivityRecord){
		func(r *budget.ActivityRecord) { r.Date = "2026-11-09" },
		func(r *budget.ActivityRecord) { r.AccountID = a.AccountID },
		func(r *budget.ActivityRecord) { r.ActivitySubtype = "withdrawal" },
		func(r *budget.ActivityRecord) { r.ActivitySubtype = "buy" },
		func(r *budget.ActivityRecord) { r.AmountCents = pointer(int64(9999)) },
		func(r *budget.ActivityRecord) { r.Currency = pointer("EUR") },
		func(r *budget.ActivityRecord) { r.Pending = true },
		func(r *budget.ActivityRecord) { r.Cancelled = true },
	}
	for _, change := range cases {
		copy := b
		change(&copy)
		if ValidTransferPair(a, copy) {
			t.Fatal("unsupported transfer pair accepted")
		}
	}
}

func TestExcludedInvestmentAccountStillSuppliesTransferEvidence(t *testing.T) {
	withdraw := investmentRecord("out", "brokerage", "2026-10-31", "cash", "withdrawal", 50000)
	deposit := investmentRecord("in", "roth", "2026-11-02", "cash", "deposit", -50000)
	sources := sourcesFor("brokerage", "roth")
	sources[0].Accounts[0].Enabled = false
	result := projectFor(t, []budget.ActivityRecord{withdraw, deposit}, sources, "household")
	if len(result.Records) != 1 || result.ReviewCount != 1 || result.Totals.NetInvestmentContributionsCents != 0 {
		t.Fatal("excluding a saved investment account created new funding")
	}
}
