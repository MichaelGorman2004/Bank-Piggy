package budget

import (
	"errors"
	"time"
)

const dateLayout = "2006-01-02"
const firstPlanningYear = 2000
const lastPlanningYear = 2100
const weeklyDays = 7
const biweeklyDays = 14
const lastDayOfMonth = 31

type payCalendar struct {
	anchor    time.Time
	frequency string
	days      []DayOfMonth
	monthEnd  bool
}

// ValidDate accepts an exact YYYY-MM-DD date from 2000 through 2100.
func ValidDate(value string) bool {
	parsed, err := time.Parse(dateLayout, value)
	return err == nil && parsed.Format(dateLayout) == value && validPlanningYear(parsed.Year())
}

func validPlanningYear(year int) bool {
	return year >= firstPlanningYear && year <= lastPlanningYear
}

func validatePaySchedule(member Member) error {
	schedule := member.PaySchedule
	if schedule == nil {
		return nil
	}
	if !ValidDate(string(schedule.AnchorDate)) {
		return errors.New("a pay schedule needs a valid date from 2000 through 2100")
	}
	if schedule.SemiMonthlyDays == nil {
		return errors.New("semimonthly days must be an array")
	}
	if err := validateMonthEndSchedule(member); err != nil {
		return err
	}
	switch member.PayFrequency {
	case "weekly", "biweekly", "monthly":
		if len(schedule.SemiMonthlyDays) != 0 {
			return errors.New("pay days are only available for semimonthly pay")
		}
		return nil
	case "semimonthly":
		return validateSemiMonthlyDays(schedule)
	default:
		return errors.New("a pay schedule needs a regular pay frequency")
	}
}

func validateMonthEndSchedule(member Member) error {
	if !member.PaySchedule.MonthEnd {
		return nil
	}
	if member.PayFrequency != "monthly" {
		return errors.New("month-end pay is only available for a monthly schedule")
	}
	anchor, err := time.Parse(dateLayout, string(member.PaySchedule.AnchorDate))
	if err != nil {
		return err
	}
	if !anchor.Equal(calendarDay(anchor, lastDayOfMonth)) {
		return errors.New("month-end pay needs an anchor on the last day of its month")
	}
	return nil
}

func validateSemiMonthlyDays(schedule *PaySchedule) error {
	days := schedule.SemiMonthlyDays
	if len(days) != 2 || days[0] < 1 || days[0] > 27 || days[1] <= days[0] || days[1] > 31 {
		return errors.New("semimonthly pay needs two increasing days from 1 to 31 that remain distinct in February")
	}
	anchor, err := time.Parse(dateLayout, string(schedule.AnchorDate))
	if err != nil {
		return err
	}
	if !anchor.Equal(calendarDay(anchor, int(days[0]))) && !anchor.Equal(calendarDay(anchor, int(days[1]))) {
		return errors.New("pay schedule anchor must be one of its semimonthly pay days")
	}
	return nil
}

func memberPayCalendar(member Member) (payCalendar, bool) {
	if member.PaySchedule == nil || validatePaySchedule(member) != nil {
		return payCalendar{}, false
	}
	anchor, err := time.Parse(dateLayout, string(member.PaySchedule.AnchorDate))
	if err != nil {
		return payCalendar{}, false
	}
	return payCalendar{anchor: anchor, frequency: member.PayFrequency, days: member.PaySchedule.SemiMonthlyDays, monthEnd: member.PaySchedule.MonthEnd}, true
}

func monthStart(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func calendarDay(month time.Time, day int) time.Time {
	start := monthStart(month)
	lastDay := start.AddDate(0, 1, -1).Day()
	return start.AddDate(0, 0, min(day, lastDay)-1)
}

func intervalPayDates(calendar payCalendar, month time.Time, interval int) []time.Time {
	start, end := monthStart(month), monthStart(month).AddDate(0, 1, 0)
	days := max(0, int(start.Sub(calendar.anchor)/(24*time.Hour)))
	first := calendar.anchor.AddDate(0, 0, ((days+interval-1)/interval)*interval)
	result := []time.Time{}
	for date := first; date.Before(end); date = date.AddDate(0, 0, interval) {
		result = append(result, date)
	}
	return result
}

func payDatesInMonth(calendar payCalendar, month time.Time) []time.Time {
	switch calendar.frequency {
	case "weekly":
		return intervalPayDates(calendar, month, weeklyDays)
	case "biweekly":
		return intervalPayDates(calendar, month, biweeklyDays)
	}
	day := calendar.anchor.Day()
	if calendar.monthEnd {
		day = lastDayOfMonth
	}
	days := []DayOfMonth{DayOfMonth(day)}
	if calendar.frequency == "semimonthly" {
		days = calendar.days
	}
	result := []time.Time{}
	for _, day := range days {
		date := calendarDay(month, int(day))
		if !date.Before(calendar.anchor) {
			result = append(result, date)
		}
	}
	return result
}

func currentPayPeriod(member Member, calendar payCalendar, date time.Time) (PayPeriod, bool) {
	if date.Before(calendar.anchor) {
		return PayPeriod{}, false
	}
	dates := []time.Time{}
	for offset := -1; offset <= 1; offset++ {
		dates = append(dates, payDatesInMonth(calendar, monthStart(date).AddDate(0, offset, 0))...)
	}
	var start time.Time
	for _, payday := range dates {
		if !payday.After(date) {
			start = payday
		} else if !start.IsZero() && validPlanningYear(payday.Year()) {
			return PayPeriod{MemberID: member.ID, MemberName: member.Name, StartDate: CalendarDate(start.Format(dateLayout)), NextPayDate: CalendarDate(payday.Format(dateLayout)), DueDate: CalendarDate(payday.AddDate(0, 0, -1).Format(dateLayout))}, true
		}
	}
	// The final period is unavailable when its next payday exceeds the date contract.
	return PayPeriod{}, false
}
