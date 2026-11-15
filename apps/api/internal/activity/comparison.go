package activity

import (
	"time"

	"bankpiggy/api/internal/budget"
)

func compareMonth(result *budget.ActivityResponse, plan budget.Household, month time.Time) error {
	comparison := plannedComparison(plan, month, result.Scope)
	comparison.BankAvailable = historyAvailable(result.Sources, result.Month, "depository")
	comparison.InvestmentAvailable = historyAvailable(result.Sources, result.Month, "investment")
	for _, record := range result.Records {
		if !countedRecord(record) {
			continue
		}
		if err := compareRecord(&comparison, record); err != nil {
			return err
		}
	}
	if err := finishComparison(&comparison, result.Totals.NetInvestmentContributionsCents); err != nil {
		return err
	}
	if result.Scope != "household" {
		scopeExpenseComparisons(&comparison, plan, result.Scope)
		scopeGoalComparisons(&comparison, plan, result.Scope)
	}
	var err error
	comparison.BankFlow, err = bankFlow(result.Totals.IncomeCents, result.Categories)
	result.Comparison = comparison
	return err
}

func compareRecord(comparison *budget.ActivityComparison, record budget.ActivityRecord) error {
	if record.Kind == "bank" && record.Category == "spending" {
		return compareSpending(comparison, record)
	}
	if record.Kind == "investment" && (record.Category == "contribution" || record.Category == "withdrawal") {
		return compareInvestment(comparison, record)
	}
	return nil
}

func compareSpending(comparison *budget.ActivityComparison, record budget.ActivityRecord) error {
	amount := *record.ScopeAmountCents
	if record.ExpenseID != nil {
		for index := range comparison.Expenses {
			row := &comparison.Expenses[index]
			if row.ExpenseID == *record.ExpenseID {
				value, err := addCents(row.SpentCents, amount)
				row.SpentCents = value
				row.RecordCount++
				return err
			}
		}
	}
	value, err := addCents(comparison.UnmatchedSpendingCents, amount)
	comparison.UnmatchedSpendingCents = value
	comparison.UnmatchedSpendingCount++
	return err
}

func compareInvestment(comparison *budget.ActivityComparison, record budget.ActivityRecord) error {
	amount := absolute(*record.ScopeAmountCents)
	if record.Category == "withdrawal" {
		amount = -amount
	}
	if record.GoalID != nil {
		for index := range comparison.InvestmentGoals {
			row := &comparison.InvestmentGoals[index]
			if row.GoalID == *record.GoalID {
				value, err := addCents(row.NetContributionsCents, amount)
				row.NetContributionsCents = value
				row.RecordCount++
				return err
			}
		}
	}
	value, err := addCents(comparison.UnmatchedInvestmentNetCents, amount)
	comparison.UnmatchedInvestmentNetCents = value
	return err
}

func finishComparison(comparison *budget.ActivityComparison, netInvestment int64) error {
	for index := range comparison.Expenses {
		row := &comparison.Expenses[index]
		value, err := addCents(row.BudgetCents, -row.SpentCents)
		if err != nil {
			return err
		}
		row.RemainingCents = value
	}
	for index := range comparison.InvestmentGoals {
		row := &comparison.InvestmentGoals[index]
		value, err := addCents(row.BudgetCents, -row.NetContributionsCents)
		if err != nil {
			return err
		}
		row.RemainingCents = value
	}
	var err error
	comparison.InvestmentBudgetCents, err = addCents(comparison.InvestmentGoalBudgetCents, comparison.PayrollBudgetCents)
	if err != nil {
		return err
	}
	comparison.InvestmentRemainingCents, err = addCents(comparison.InvestmentBudgetCents, -netInvestment)
	return err
}
