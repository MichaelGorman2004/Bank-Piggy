// Package activity derives reviewable actuals from separately stored provider history.
package activity

import (
	"math"
	"strings"
	"time"

	"bankpiggy/api/internal/budget"
)

const maxSafeCents int64 = 9_007_199_254_740_991

var spendingCategories = map[string]bool{
	"BANK_FEES": true, "ENTERTAINMENT": true, "FOOD_AND_DRINK": true,
	"GENERAL_MERCHANDISE": true, "GENERAL_SERVICES": true,
	"GOVERNMENT_AND_NON_PROFIT": true, "HOME_IMPROVEMENT": true,
	"MEDICAL": true, "PERSONAL_CARE": true, "RENT_AND_UTILITIES": true,
	"TRANSPORTATION": true, "TRAVEL": true,
}

func reason(record budget.ActivityRecord, category budget.ActivityCategory, message string) budget.ActivityRecord {
	record.Category = category
	record.ReviewReason = &message
	return record
}

// Classify preserves explicit reviews and excludes facts that cannot form posted USD actuals.
func Classify(record budget.ActivityRecord) budget.ActivityRecord {
	record.TransferCandidates = []budget.ActivityTransferCandidate{}
	if record.Cancelled || record.Pending {
		return reason(record, "excluded", "Only posted, uncancelled activity enters totals.")
	}
	if !usableCents(record.AmountCents) {
		record.AmountCents = nil
		return reason(record, "review", "A usable whole-cent amount was not provided.")
	}
	if record.Currency == nil || *record.Currency != "USD" {
		return reason(record, "excluded", "This currency is not included in USD totals.")
	}
	if _, err := time.Parse(time.DateOnly, string(record.Date)); err != nil {
		return reason(record, "review", "A valid posting date was not provided.")
	}
	if record.IsReviewed {
		return record
	}
	record.ReviewReason = nil
	if record.Kind == "investment" {
		record.Category = investmentCategory(record)
	} else {
		record.Category = bankCategory(record)
	}
	if record.Category == "review" {
		return reason(record, "review", "Check what this transaction represents before including it.")
	}
	return record
}

func bankCategory(record budget.ActivityRecord) budget.ActivityCategory {
	primary := strings.ToUpper(record.PrimaryCategory)
	if primary == "TRANSFER_IN" || primary == "TRANSFER_OUT" || primary == "LOAN_PAYMENTS" {
		return "transfer"
	}
	if record.CategoryConfidence != "HIGH" && record.CategoryConfidence != "VERY_HIGH" {
		return "review"
	}
	if primary == "INCOME" {
		if record.DetailedCategory == "INCOME_WAGES" || record.DetailedCategory == "INCOME_SALARY" {
			return "paycheck"
		}
		return "income"
	}
	if spendingCategories[primary] {
		return "spending"
	}
	return "review"
}

func investmentCategory(record budget.ActivityRecord) budget.ActivityCategory {
	subtype := strings.ToLower(record.ActivitySubtype)
	switch subtype {
	case "contribution", "deposit":
		return "contribution"
	case "withdrawal", "distribution":
		return "withdrawal"
	case "dividend", "qualified dividend", "non-qualified dividend", "interest", "long-term capital gain", "short-term capital gain":
		return "investment_income"
	case "dividend reinvestment", "interest reinvestment", "long-term capital gain reinvestment", "short-term capital gain reinvestment":
		return "trade"
	}
	switch record.ActivityType {
	case "buy", "sell":
		return "trade"
	case "transfer":
		return "transfer"
	case "fee":
		return "spending"
	case "cancel":
		return "excluded"
	default:
		return "review"
	}
}

func absolute(cents int64) int64 {
	if cents == math.MinInt64 {
		return maxSafeCents
	}
	return max(cents, -cents)
}

func usableCents(cents *int64) bool {
	return cents != nil && *cents <= maxSafeCents && *cents >= -maxSafeCents
}
