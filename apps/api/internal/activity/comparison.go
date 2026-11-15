package activity

import (
	"time"

	"bankpiggy/api/internal/budget"
)

func compareMonth(result *budget.ActivityResponse, plan budget.Household, month time.Time) error {
	comparison, err := plannedComparison(plan, month, result.Scope)
	if err != nil {
		return err
	}
	comparison.BankAvailable = historyAvailable(result.Sources, result.Month, "depository")
	comparison.InvestmentAvailable = historyAvailable(result.Sources, result.Month, "investment")
	comparison.ExpenseRemainingCents, err = addCents(comparison.ExpenseBudgetCents, -result.Totals.SpendingCents)
	if err != nil {
		return err
	}
	comparison.InvestmentRemainingCents, err = addCents(comparison.InvestmentBudgetCents, -result.Totals.NetInvestmentContributionsCents)
	if err != nil {
		return err
	}
	comparison.BankFlow, err = bankFlow(result.Totals.IncomeCents, result.Totals.SpendingCents)
	result.Comparison = comparison
	return err
}
