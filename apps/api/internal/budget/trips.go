package budget

import "errors"

func validateTrip(trip *TripBudget) error {
	if trip == nil {
		return nil
	}
	if !validMoney(trip.SavedAmountCents) || !ValidMonth(trip.TargetMonth) {
		return errors.New("trip savings or target month is invalid")
	}
	if len(trip.Costs) < 1 || len(trip.Costs) > 6 {
		return errors.New("trip needs one to six distinct budget categories")
	}
	return validateTripCosts(trip.Costs)
}

func validTripCategory(category string) bool {
	switch category {
	case "flights", "lodging", "food", "transport", "activities", "other":
		return true
	}
	return false
}

func validateTripCosts(costs []TripCost) error {
	seen := map[string]bool{}
	var total int64
	for _, cost := range costs {
		if !validTripCategory(cost.Category) || seen[cost.Category] || !validMoney(cost.AmountCents) {
			return errors.New("trip costs need distinct valid categories and whole-cent amounts")
		}
		seen[cost.Category] = true
		total += cost.AmountCents
	}
	if total > MaxMoneyCents {
		return errors.New("trip budget exceeds the maximum amount")
	}
	return nil
}

func tripProjection(goal Goal, month string) GoalProjection {
	var target int64
	for _, cost := range goal.Trip.Costs {
		target += cost.AmountCents
	}
	return targetProjection(goal.ID, target, goal.Trip.SavedAmountCents, goal.Trip.TargetMonth, month)
}

func tripProjections(goals []Goal, month string) []GoalProjection {
	result := []GoalProjection{}
	for _, goal := range goals {
		if goal.Trip != nil {
			result = append(result, tripProjection(goal, month))
		}
	}
	return result
}
