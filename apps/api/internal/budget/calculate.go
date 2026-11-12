package budget

import "math/big"

// proportional rounds the first share to the nearest cent; ties go to member one.
// big.Int prevents overflow when large salary and expense inputs are multiplied.
func proportional(amount, numerator, denominator int64) int64 {
	value := new(big.Int).Mul(big.NewInt(amount), big.NewInt(numerator))
	value.Add(value, big.NewInt(denominator/2))
	return value.Quo(value, big.NewInt(denominator)).Int64()
}

func splitExpense(h Household, expense Expense) (int64, int64) {
	amount := expense.MonthlyAmountCents
	if len(h.Members) == 1 || expense.OwnerID == h.Members[0].ID {
		return amount, 0
	}
	if expense.OwnerID == h.Members[1].ID {
		return 0, amount
	}
	first := (amount + 1) / 2
	switch expense.Split.Method {
	case "custom":
		first = proportional(amount, expense.Split.CustomFirstShareBps, 10000)
	case "salary":
		first = proportional(amount, h.Members[0].AnnualBaseSalaryCents, h.Members[0].AnnualBaseSalaryCents+h.Members[1].AnnualBaseSalaryCents)
	}
	return first, amount - first
}

func finalSummary(summary Summary) Summary {
	summary.MonthlyAvailableCents = summary.MonthlyTakeHomeCents - summary.MonthlyExpensesCents
	summary.MonthlyRemainingCents = summary.MonthlyAvailableCents - summary.MonthlyGoalsCents
	return summary
}

func combined(first, second Summary) Summary {
	return finalSummary(Summary{
		MonthlyTakeHomeCents: first.MonthlyTakeHomeCents + second.MonthlyTakeHomeCents,
		MonthlyExpensesCents: first.MonthlyExpensesCents + second.MonthlyExpensesCents,
		MonthlyGoalsCents:    first.MonthlyGoalsCents + second.MonthlyGoalsCents,
		SavingsCents:         first.SavingsCents + second.SavingsCents,
		InvestmentsCents:     first.InvestmentsCents + second.InvestmentsCents,
		DebtCents:            first.DebtCents + second.DebtCents,
	})
}

func memberSummaries(h Household, count int64) []Summary {
	personal := make([]Summary, MaxMembers)
	for index, member := range h.Members {
		savings, investments := accountTotals(member.Accounts)
		personal[index] = Summary{MonthlyTakeHomeCents: member.MonthlyTakeHomeCents * count, SavingsCents: savings, InvestmentsCents: investments, DebtCents: member.DebtCents}
	}
	return personal
}

func expensePeriods(expense Expense, months []string) int64 {
	var count int64
	for _, month := range months {
		if activeExpense(expense, month) {
			count++
		}
	}
	return count
}

func addExpenses(h Household, months []string, personal []Summary) []ExpenseShare {
	shares := make([]ExpenseShare, 0, len(h.Expenses))
	for _, expense := range h.Expenses {
		first, second := splitExpense(h, expense)
		count := expensePeriods(expense, months)
		first, second = first*count, second*count
		personal[0].MonthlyExpensesCents += first
		personal[1].MonthlyExpensesCents += second
		shares = append(shares, ExpenseShare{expense.ID, first, second})
	}
	return shares
}

func addGoals(h Household, count int64, personal []Summary) []GoalShare {
	shares := make([]GoalShare, 0, len(h.Goals))
	for _, goal := range h.Goals {
		first, second := splitExpense(h, Expense{OwnerID: goal.OwnerID, MonthlyAmountCents: goal.MonthlyAmountCents, Split: Split{Method: "equal"}})
		first, second = first*count, second*count
		shares = append(shares, GoalShare{goal.ID, first, second})
		personal[0].MonthlyGoalsCents += first
		personal[1].MonthlyGoalsCents += second
	}
	return shares
}

func calculateMonths(h Household, months []string) BudgetSummary {
	personal := memberSummaries(h, int64(len(months)))
	result := BudgetSummary{
		Members:       map[string]Summary{},
		ExpenseShares: addExpenses(h, months, personal),
		GoalShares:    addGoals(h, int64(len(months)), personal),
	}
	for index, member := range h.Members {
		result.Members[member.ID] = finalSummary(personal[index])
	}
	result.Household = combined(personal[0], personal[1])
	return result
}
