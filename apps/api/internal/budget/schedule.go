package budget

import (
	"errors"
	"time"
)

const monthLayout = "2006-01"

// ValidMonth accepts an exact YYYY-MM calendar month with a positive year.
func ValidMonth(value string) bool {
	parsed, err := time.Parse(monthLayout, value)
	return err == nil && parsed.Year() > 0 && parsed.Format(monthLayout) == value
}

func monthNumber(value string) int64 {
	parsed, err := time.Parse(monthLayout, value)
	if err != nil {
		return 0
	}
	return int64(parsed.Year()*12 + int(parsed.Month()))
}

func activeExpense(expense Expense, month string) bool {
	return (expense.StartMonth == "" || expense.StartMonth <= month) &&
		(expense.EndMonth == "" || expense.EndMonth >= month)
}

func validPaymentKind(kind string) bool {
	switch kind {
	case "", "subscription", "bill", "loan":
		return true
	}
	return false
}

func validateSchedule(expense Expense) error {
	if !validPaymentKind(expense.Kind) {
		return errors.New("invalid recurring payment kind")
	}
	if expense.ProviderID != "" && !validProvider.MatchString(expense.ProviderID) {
		return errors.New("invalid payment provider")
	}
	for _, month := range []string{expense.StartMonth, expense.EndMonth} {
		if month != "" && !ValidMonth(month) {
			return errors.New("payment dates must be valid YYYY-MM months")
		}
	}
	if expense.StartMonth != "" && expense.EndMonth != "" && expense.EndMonth < expense.StartMonth {
		return errors.New("payment end month must be on or after its start month")
	}
	return nil
}

// Response calculates one month and the following twelve-month projection.
// The validated month is supplied by the server, so read and save use one period.
func Response(h Household, start time.Time) HouseholdResponse {
	month := start.Format(monthLayout)
	months := make([]string, 12)
	for index := range months {
		months[index] = start.AddDate(0, index, 0).Format(monthLayout)
	}
	return HouseholdResponse{
		Household: h, PlanningMonth: month,
		Summary: calculateMonths(h, months[:1]), YearlySummary: calculateMonths(h, months),
		TripProjections: tripProjections(h.Goals, month), PurchaseProjections: purchaseProjections(h.Goals, month),
	}
}
