package budget

import "errors"

func validateGoalTarget(goal Goal) error {
	var kinds int
	for _, present := range []bool{goal.Trip != nil, goal.Purchase != nil, goal.SavingsTarget != nil} {
		if present {
			kinds++
		}
	}
	if kinds > 1 {
		return errors.New("ordinary savings, trip, and purchase progress must be mutually exclusive")
	}
	if err := validateTrip(goal.Trip); err != nil {
		return err
	}
	if err := validatePurchase(goal.Purchase); err != nil {
		return err
	}
	return validateSavingsTarget(goal.SavingsTarget)
}

func validatePurchase(purchase *PurchaseTarget) error {
	if purchase == nil {
		return nil
	}
	if !validMoney(purchase.TargetAmountCents) || !validMoney(purchase.SavedAmountCents) || !ValidMonth(purchase.TargetMonth) {
		return errors.New("purchase target, savings, or target month is invalid")
	}
	return nil
}

func validateSavingsTarget(target *SavingsTarget) error {
	if target == nil {
		return nil
	}
	if target.TargetAmountCents <= 0 || !validMoney(target.TargetAmountCents) || !validMoney(target.SavedAmountCents) {
		return errors.New("savings target must be positive whole cents with nonnegative saved progress")
	}
	return nil
}

// targetProjection uses validated cents and months without changing budget allocations.
// The planning month and target month both count as monthly saving periods.
func targetProjection(goalID string, target, saved int64, targetMonth, month string) GoalProjection {
	result := GoalProjection{
		GoalID: goalID, TargetAmountCents: target,
		RemainingAmountCents: max(0, target-saved),
		MonthsRemaining:      max(0, monthNumber(targetMonth)-monthNumber(month)+1),
		Status:               "saving",
	}
	if result.RemainingAmountCents == 0 {
		result.Status = "funded"
	} else if result.MonthsRemaining == 0 {
		result.Status = "overdue"
	} else {
		result.SuggestedMonthlyCents = (result.RemainingAmountCents + result.MonthsRemaining - 1) / result.MonthsRemaining
	}
	return result
}

func purchaseProjections(goals []Goal, month string) []GoalProjection {
	result := []GoalProjection{}
	for _, goal := range goals {
		if target := goal.Purchase; target != nil {
			result = append(result, targetProjection(goal.ID, target.TargetAmountCents, target.SavedAmountCents, target.TargetMonth, month))
		}
	}
	return result
}
