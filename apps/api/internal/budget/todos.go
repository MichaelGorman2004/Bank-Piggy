package budget

import (
	"cmp"
	"slices"
	"time"
)

// Todos builds the current pay-period checklist from a validated plan and date.
// Each task starts incomplete; acknowledging it never changes saved progress.
func Todos(h Household, onDate time.Time) TodoList {
	date := time.Date(onDate.Year(), onDate.Month(), onDate.Day(), 0, 0, 0, 0, time.UTC)
	result := TodoList{Date: CalendarDate(date.Format(dateLayout)), PlanRevision: h.Revision, Periods: []PayPeriod{}, Tasks: []TodoItem{}, UnscheduledExpenseIds: unscheduledExpenses(h.Expenses, date)}
	if !validPlanningYear(date.Year()) {
		return result
	}
	for index, member := range h.Members {
		calendar, ok := memberPayCalendar(member)
		if !ok {
			continue
		}
		period, ok := currentPayPeriod(member, calendar, date)
		if !ok {
			continue
		}
		result.Periods = append(result.Periods, period)
		result.Tasks = append(result.Tasks, goalTodos(h, index, calendar, period)...)
		result.Tasks = append(result.Tasks, expenseTodos(h, index, period)...)
	}
	slices.SortFunc(result.Tasks, compareTodos)
	return result
}

func compareTodos(first, second TodoItem) int {
	for _, pair := range [][2]string{{string(first.DueDate), string(second.DueDate)}, {first.MemberID, second.MemberID}, {first.SourceID, second.SourceID}, {first.Kind, second.Kind}} {
		if order := cmp.Compare(pair[0], pair[1]); order != 0 {
			return order
		}
	}
	return 0
}

func newTodo(memberID, sourceID, kind, name string, amount int64, period PayPeriod, dueDate CalendarDate) TodoItem {
	identityDate := period.StartDate
	if kind == "expense" {
		identityDate = dueDate
	}
	return TodoItem{
		ID:       memberID + ":" + kind + ":" + sourceID + ":" + string(identityDate),
		MemberID: memberID, SourceID: sourceID, Kind: kind, Name: name,
		AmountCents: amount, PeriodStart: period.StartDate, DueDate: dueDate,
	}
}

func personalGoalAmount(h Household, index int, goal Goal) int64 {
	first, second := splitExpense(h, Expense{OwnerID: goal.OwnerID, MonthlyAmountCents: goal.MonthlyAmountCents, Split: Split{Method: "equal"}})
	if index == 0 {
		return first
	}
	return second
}

// goalPaydayAmount assigns remainder cents to the earliest actual paydays in
// the start month. Those paydays together equal the member's monthly allocation.
func goalPaydayAmount(amount int64, calendar payCalendar, start time.Time) int64 {
	dates := payDatesInMonth(calendar, start)
	count := int64(len(dates))
	for index, date := range dates {
		if date.Equal(start) {
			share := amount / count
			if int64(index) < amount%count {
				share++
			}
			return share
		}
	}
	return 0
}

func goalTodos(h Household, index int, calendar payCalendar, period PayPeriod) []TodoItem {
	start, err := time.Parse(dateLayout, string(period.StartDate))
	if err != nil {
		return nil
	}
	result := []TodoItem{}
	for _, goal := range h.Goals {
		if goal.ContributionYear != nil && int(*goal.ContributionYear) != start.Year() {
			continue
		}
		amount := goalPaydayAmount(personalGoalAmount(h, index, goal), calendar, start)
		if goal.ContributionYear != nil && goal.SavingsTarget != nil {
			amount = min(amount, max(0, goal.SavingsTarget.TargetAmountCents-goal.SavingsTarget.SavedAmountCents))
		}
		if amount > 0 {
			result = append(result, newTodo(period.MemberID, goal.ID, "goal", goal.Name, amount, period, period.DueDate))
		}
	}
	return result
}
