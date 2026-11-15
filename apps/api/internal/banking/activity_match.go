package banking

import "bankpiggy/api/internal/budget"

// activityMatchAllocation captures reviewed cost shares without changing the plan.
// Keeping the same target preserves its allocation; clearing and rematching recalculates it.
func activityMatchAllocation(plan budget.Household, record budget.ActivityRecord, request budget.ActivityReviewRequest) (*budget.ActivityAllocation, error) {
	if request.ExpenseID == nil && request.GoalID == nil {
		return nil, nil
	}
	if request.Category == nil || request.TransferPeerID != nil || record.AmountCents == nil {
		return nil, ErrInvalid
	}
	if request.ExpenseID != nil && request.GoalID != nil {
		return nil, ErrInvalid
	}
	if request.ExpenseID != nil {
		return expenseMatchAllocation(plan, record, request)
	}
	return goalMatchAllocation(plan, record, request)
}

func expenseMatchAllocation(plan budget.Household, record budget.ActivityRecord, request budget.ActivityReviewRequest) (*budget.ActivityAllocation, error) {
	if record.Kind != "bank" || *request.Category != "spending" {
		return nil, ErrInvalid
	}
	for _, expense := range plan.Expenses {
		if expense.ID != *request.ExpenseID {
			continue
		}
		if record.ExpenseID != nil && *record.ExpenseID == expense.ID && record.Allocation != nil {
			return record.Allocation, nil
		}
		expense.MonthlyAmountCents = *record.AmountCents
		first, _ := budget.SplitExpense(plan, expense)
		return &budget.ActivityAllocation{Ownership: expense.OwnerID, FirstMemberCents: first}, nil
	}
	return nil, ErrNotFound
}

func goalMatchAllocation(plan budget.Household, record budget.ActivityRecord, request budget.ActivityReviewRequest) (*budget.ActivityAllocation, error) {
	if record.Kind != "investment" || (*request.Category != "contribution" && *request.Category != "withdrawal") {
		return nil, ErrInvalid
	}
	for _, goal := range plan.Goals {
		if goal.ID != *request.GoalID {
			continue
		}
		if !budget.IsInvestmentGoal(goal) {
			return nil, ErrInvalid
		}
		if record.GoalID != nil && *record.GoalID == goal.ID && record.Allocation != nil {
			return record.Allocation, nil
		}
		first, _ := budget.SplitExpense(plan, budget.Expense{OwnerID: goal.OwnerID, MonthlyAmountCents: *record.AmountCents, Split: budget.Split{Method: "equal"}})
		return &budget.ActivityAllocation{Ownership: goal.OwnerID, FirstMemberCents: first}, nil
	}
	return nil, ErrNotFound
}
