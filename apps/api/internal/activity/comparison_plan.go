package activity

import (
	"time"

	"bankpiggy/api/internal/budget"
)

func plannedComparison(plan budget.Household, month time.Time, scope string) (budget.ActivityComparison, error) {
	response := budget.Response(plan, month)
	var result budget.ActivityComparison
	var err error
	for _, share := range response.Summary.ExpenseShares {
		amount := selectedShare(plan, scope, share.FirstMemberCents, share.SecondMemberCents)
		result.ExpenseBudgetCents, err = addCents(result.ExpenseBudgetCents, amount)
		if err != nil {
			return result, err
		}
	}
	for index, goal := range plan.Goals {
		if !budget.IsInvestmentGoal(goal) {
			continue
		}
		share := response.Summary.GoalShares[index]
		amount := selectedShare(plan, scope, share.FirstMemberCents, share.SecondMemberCents)
		result.InvestmentGoalBudgetCents, err = addCents(result.InvestmentGoalBudgetCents, amount)
		if err != nil {
			return result, err
		}
	}
	for _, row := range response.PayrollSummaries {
		if scope == "household" || scope == row.MemberID {
			amount := row.Traditional401KMonthlyCents + row.Roth401KMonthlyCents + row.EmployerMonthlyCents
			result.PayrollBudgetCents, err = addCents(result.PayrollBudgetCents, amount)
			if err != nil {
				return result, err
			}
		}
	}
	result.InvestmentBudgetCents, err = addCents(result.InvestmentGoalBudgetCents, result.PayrollBudgetCents)
	return result, err
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
