package budget

import (
	"errors"
	"slices"
	"time"
)

func validatePaymentTaskFields(expense Expense) error {
	if expense.PaymentMode != "" && expense.PaymentMode != "automatic" && expense.PaymentMode != "manual" {
		return errors.New("invalid payment mode")
	}
	if expense.DueDay != nil && (*expense.DueDay < 1 || *expense.DueDay > 31) {
		return errors.New("payment due day must be between 1 and 31")
	}
	return nil
}

func manualPayment(expense Expense) bool {
	if expense.PaymentMode != "" {
		return expense.PaymentMode == "manual"
	}
	return expense.Kind != "subscription"
}

func unscheduledExpenses(expenses []Expense, date time.Time) []string {
	result := []string{}
	month := date.Format(monthLayout)
	for _, expense := range expenses {
		if manualPayment(expense) && expense.DueDay == nil && activeExpense(expense, month) {
			result = append(result, expense.ID)
		}
	}
	slices.Sort(result)
	return result
}

func personalPaymentAmount(h Household, index int, expense Expense) (int64, string) {
	if expense.PaidByMemberID != "" {
		if expense.PaidByMemberID == h.Members[index].ID {
			return expense.MonthlyAmountCents, expense.Name
		}
		return 0, expense.Name
	}
	name := expense.Name
	if expense.OwnerID == "shared" && len(h.Members) > 1 {
		name += " (your share)"
	}
	first, second := splitExpense(h, expense)
	if index == 0 {
		return first, name
	}
	return second, name
}

func expenseTaskInMonth(expense Expense, month, start, end time.Time) (CalendarDate, bool) {
	if !activeExpense(expense, month.Format(monthLayout)) {
		return "", false
	}
	if expense.DueDay == nil {
		return "", false
	}
	due := calendarDay(month, int(*expense.DueDay))
	return CalendarDate(due.Format(dateLayout)), !due.Before(start) && due.Before(end)
}

func expenseTodos(h Household, index int, period PayPeriod) []TodoItem {
	start, startErr := time.Parse(dateLayout, string(period.StartDate))
	end, endErr := time.Parse(dateLayout, string(period.NextPayDate))
	if startErr != nil || endErr != nil {
		return nil
	}
	result := []TodoItem{}
	for _, expense := range h.Expenses {
		amount, name := personalPaymentAmount(h, index, expense)
		if !manualPayment(expense) || amount <= 0 {
			continue
		}
		for month := monthStart(start); month.Before(end); month = month.AddDate(0, 1, 0) {
			if due, included := expenseTaskInMonth(expense, month, start, end); included {
				result = append(result, newTodo(period.MemberID, expense.ID, "expense", name, amount, period, due))
			}
		}
	}
	return result
}
