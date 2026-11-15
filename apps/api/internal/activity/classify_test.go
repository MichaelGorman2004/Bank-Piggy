package activity

import (
	"math"
	"testing"

	"bankpiggy/api/internal/budget"
)

func pointer[T any](value T) *T { return &value }

func bankRecord(id, date string, amount int64, primary, detailed string) budget.ActivityRecord {
	return budget.ActivityRecord{ID: id, AccountID: "checking", Kind: "bank", Date: budget.CalendarDate(date), AmountCents: &amount, Currency: pointer("USD"), PrimaryCategory: primary, DetailedCategory: detailed, CategoryConfidence: "HIGH", Revision: 1}
}

func investmentRecord(id, account, date, kind, subtype string, amount int64) budget.ActivityRecord {
	return budget.ActivityRecord{ID: id, AccountID: account, AccountName: account, Kind: "investment", Date: budget.CalendarDate(date), AmountCents: &amount, Currency: pointer("USD"), ActivityType: kind, ActivitySubtype: subtype, Revision: 1}
}

func TestClassificationKeepsFundingSeparateFromTradesAndBankActuals(t *testing.T) {
	cases := []struct {
		record   budget.ActivityRecord
		category budget.ActivityCategory
	}{
		{bankRecord("pay", "2026-11-01", -120000, "INCOME", "INCOME_WAGES"), "paycheck"},
		{bankRecord("pay-v2", "2026-11-01", -120000, "INCOME", "INCOME_SALARY"), "paycheck"},
		{bankRecord("income", "2026-11-01", -300, "INCOME", "INCOME_INTEREST_EARNED"), "income"},
		{bankRecord("purchase", "2026-11-01", 1500, "FOOD_AND_DRINK", ""), "spending"},
		{bankRecord("refund", "2026-11-01", -500, "FOOD_AND_DRINK", ""), "spending"},
		{bankRecord("card", "2026-11-01", 5000, "LOAN_PAYMENTS", "LOAN_PAYMENTS_CREDIT_CARD_PAYMENT"), "transfer"},
		{bankRecord("move", "2026-11-01", 5000, "TRANSFER_OUT", ""), "transfer"},
		{bankRecord("unknown", "2026-11-01", 5000, "OTHER", ""), "review"},
		{investmentRecord("combined", "roth", "2026-11-01", "buy", "contribution", 50000), "contribution"},
		{investmentRecord("deposit", "roth", "2026-11-01", "cash", "deposit", -50000), "contribution"},
		{investmentRecord("withdraw", "roth", "2026-11-01", "cash", "withdrawal", 2000), "withdrawal"},
		{investmentRecord("buy", "roth", "2026-11-01", "buy", "buy", 5000), "trade"},
		{investmentRecord("sell", "roth", "2026-11-01", "sell", "sell", -5000), "trade"},
		{investmentRecord("dividend", "roth", "2026-11-01", "cash", "dividend", -50), "investment_income"},
		{investmentRecord("reinvest", "roth", "2026-11-01", "buy", "dividend reinvestment", 50), "trade"},
		{investmentRecord("fee", "roth", "2026-11-01", "fee", "management fee", 200), "spending"},
		{investmentRecord("move", "roth", "2026-11-01", "transfer", "transfer", -500), "transfer"},
	}
	for _, tc := range cases {
		if got := Classify(tc.record); got.Category != tc.category {
			t.Errorf("%s: got %s, want %s", tc.record.ID, got.Category, tc.category)
		}
	}
}

func TestUnsupportedFactsCannotBeReviewedIntoTotals(t *testing.T) {
	base := bankRecord("x", "2026-11-01", 123, "FOOD_AND_DRINK", "")
	cases := []struct {
		name     string
		change   func(*budget.ActivityRecord)
		category budget.ActivityCategory
	}{
		{"pending", func(r *budget.ActivityRecord) { r.Pending = true }, "excluded"},
		{"cancelled", func(r *budget.ActivityRecord) { r.Cancelled = true }, "excluded"},
		{"currency", func(r *budget.ActivityRecord) { r.Currency = pointer("CAD") }, "excluded"},
		{"missing amount", func(r *budget.ActivityRecord) { r.AmountCents = nil }, "review"},
		{"overflow", func(r *budget.ActivityRecord) { r.AmountCents = pointer(int64(math.MinInt64)) }, "review"},
	}
	for _, tc := range cases {
		r := base
		tc.change(&r)
		if got := Classify(r); got.Category != tc.category || ValidReview(r, "income") {
			t.Errorf("%s was counted or reviewable", tc.name)
		}
	}
	base.IsReviewed, base.Category = true, "excluded"
	if Classify(base).Category != "excluded" {
		t.Fatal("explicit exclusion was discarded")
	}
	base.CategoryConfidence = "LOW"
	base.IsReviewed = false
	if Classify(base).Category != "review" {
		t.Fatal("low-confidence category entered totals")
	}
	for _, confidence := range []string{"", "UNKNOWN", "MEDIUM", "unrecognized"} {
		base.CategoryConfidence = confidence
		if Classify(base).Category != "review" {
			t.Fatal("unconfirmed category entered totals")
		}
	}
}
