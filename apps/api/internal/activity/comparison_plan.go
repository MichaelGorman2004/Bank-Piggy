package activity

import (
	"time"

	"bankpiggy/api/internal/budget"
)

func plannedComparison(plan budget.Household, month time.Time, scope string) budget.ActivityComparison {
	response := budget.Response(plan, month)
	result := budget.ActivityComparison{
		Expenses: []budget.ActivityExpenseComparison{}, InvestmentGoals: []budget.ActivityGoalComparison{},
	}
	for index, expense := range plan.Expenses {
		share := response.Summary.ExpenseShares[index]
		result.Expenses = append(result.Expenses, budget.ActivityExpenseComparison{
			ExpenseID: expense.ID, Name: expense.Name, Category: expense.Category,
			BudgetCents: selectedShare(plan, scope, share.FirstMemberCents, share.SecondMemberCents),
		})
	}
	for index, goal := range plan.Goals {
		if !budget.IsInvestmentGoal(goal) {
			continue
		}
		share := response.Summary.GoalShares[index]
		amount := selectedShare(plan, scope, share.FirstMemberCents, share.SecondMemberCents)
		result.InvestmentGoals = append(result.InvestmentGoals, budget.ActivityGoalComparison{
			GoalID: goal.ID, Name: goal.Name, BudgetCents: amount,
		})
		result.InvestmentGoalBudgetCents += amount
	}
	for _, row := range response.PayrollSummaries {
		if scope == "household" || scope == row.MemberID {
			result.PayrollBudgetCents += row.Traditional401KMonthlyCents + row.Roth401KMonthlyCents + row.EmployerMonthlyCents
		}
	}
	return result
}

func selectedShare(plan budget.Household, scope string, first, second int64) int64 {
	if scope == "household" {
		return first + second
	}
	if scope == plan.Members[0].ID {
		return first
	}
	return second
}

func scopeExpenseComparisons(comparison *budget.ActivityComparison, plan budget.Household, scope string) {
	rows := comparison.Expenses[:0]
	for index, row := range comparison.Expenses {
		if inScope(plan.Expenses[index].OwnerID, scope) || row.BudgetCents != 0 || row.RecordCount > 0 {
			rows = append(rows, row)
		}
	}
	comparison.Expenses = rows
}

func scopeGoalComparisons(comparison *budget.ActivityComparison, plan budget.Household, scope string) {
	owners := make(map[string]string, len(plan.Goals))
	for _, goal := range plan.Goals {
		owners[goal.ID] = goal.OwnerID
	}
	rows := comparison.InvestmentGoals[:0]
	for _, row := range comparison.InvestmentGoals {
		if inScope(owners[row.GoalID], scope) || row.BudgetCents != 0 || row.RecordCount > 0 {
			rows = append(rows, row)
		}
	}
	comparison.InvestmentGoals = rows
}

// historyAvailable distinguishes a recorded zero from missing monthly history.
// Household account selection is intentional: the account payer can differ from a cost share.
func historyAvailable(sources []budget.ActivitySource, month string, kind budget.ConnectionKind) bool {
	for _, source := range sources {
		if !availableHistorySource(source, kind) || !historyOverlaps(source, month) {
			continue
		}
		for _, account := range source.Accounts {
			if account.Enabled && account.Available {
				return true
			}
		}
	}
	return false
}

func availableHistorySource(source budget.ActivitySource, kind budget.ConnectionKind) bool {
	return source.Kind == kind && source.Enabled && source.Allowed && !source.NeedsConsent && source.State != "disabled" && source.State != "unsupported"
}

func historyOverlaps(source budget.ActivitySource, month string) bool {
	if source.HistoryStart == nil || source.HistoryEnd == nil {
		return false
	}
	start, err := time.Parse(time.DateOnly, *source.HistoryStart)
	if err != nil {
		return false
	}
	end, err := time.Parse(time.DateOnly, *source.HistoryEnd)
	return err == nil && !end.Before(start) && start.Format("2006-01") <= month && end.Format("2006-01") >= month
}
