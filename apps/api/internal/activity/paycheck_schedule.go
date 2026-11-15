package activity

import (
	"time"

	"bankpiggy/api/internal/budget"
)

const paycheckWeeklyDays = 7
const paycheckBiweeklyDays = 14

type paycheckCalendar struct {
	frequency string
	schedule  budget.PaySchedule
	next      time.Time
}

func inferPaycheckCalendar(dates []time.Time) (paycheckCalendar, bool) {
	candidates := []paycheckCalendar{}
	for _, interval := range []struct {
		frequency string
		days, min int
	}{{"weekly", paycheckWeeklyDays, 3}, {"biweekly", paycheckBiweeklyDays, 4}} {
		if calendar, ok := intervalPaycheckCalendar(dates, interval.frequency, interval.days, interval.min); ok {
			candidates = append(candidates, calendar)
		}
	}
	if calendar, ok := monthlyPaycheckCalendar(dates); ok {
		candidates = append(candidates, calendar)
	}
	if calendar, ok := semiMonthlyPaycheckCalendar(dates); ok {
		candidates = append(candidates, calendar)
	}
	if len(candidates) != 1 {
		return paycheckCalendar{}, false
	}
	return candidates[0], true
}

func intervalPaycheckCalendar(dates []time.Time, frequency string, days, minimum int) (paycheckCalendar, bool) {
	if len(dates) < minimum {
		return paycheckCalendar{}, false
	}
	for index := 1; index < len(dates); index++ {
		if !dates[index].Equal(dates[index-1].AddDate(0, 0, days)) {
			return paycheckCalendar{}, false
		}
	}
	last := dates[len(dates)-1]
	return paycheckCalendar{frequency: frequency, schedule: paycheckSchedule(last), next: last.AddDate(0, 0, days)}, true
}

func paycheckSchedule(anchor time.Time) budget.PaySchedule {
	return budget.PaySchedule{AnchorDate: budget.CalendarDate(anchor.Format(paycheckDateLayout)), SemiMonthlyDays: []budget.DayOfMonth{}}
}

func monthlyPaycheckCalendar(dates []time.Time) (paycheckCalendar, bool) {
	if len(dates) < 3 {
		return paycheckCalendar{}, false
	}
	day, monthEnd := 0, true
	for index, date := range dates {
		day = max(day, date.Day())
		monthEnd = monthEnd && date.Equal(paycheckCalendarDay(date, 31))
		if index > 0 && paycheckMonthNumber(date)-paycheckMonthNumber(dates[index-1]) != 1 {
			return paycheckCalendar{}, false
		}
	}
	if monthEnd {
		day = 31
	}
	for _, date := range dates {
		if !date.Equal(paycheckCalendarDay(date, day)) {
			return paycheckCalendar{}, false
		}
	}
	last := dates[len(dates)-1]
	schedule := paycheckSchedule(monthlyPaycheckAnchor(dates, day, monthEnd))
	schedule.MonthEnd = monthEnd
	return paycheckCalendar{frequency: "monthly", schedule: schedule, next: paycheckCalendarDay(paycheckNextMonth(last), day)}, true
}

func monthlyPaycheckAnchor(dates []time.Time, day int, monthEnd bool) time.Time {
	// A February payday can be clamped to 28. Keep an observed day-30 anchor
	// so the canonical schedule resumes on day 30 in March.
	for index := len(dates) - 1; index >= 0; index-- {
		if monthEnd || dates[index].Day() == day {
			return dates[index]
		}
	}
	return dates[len(dates)-1]
}

func semiMonthlyPaycheckCalendar(dates []time.Time) (paycheckCalendar, bool) {
	if len(dates) < 4 || paycheckMonthNumber(dates[len(dates)-1]) == paycheckMonthNumber(dates[0]) {
		return paycheckCalendar{}, false
	}
	candidates := []paycheckCalendar{}
	for first := 1; first <= 27; first++ {
		for second := first + 1; second <= 31; second++ {
			if matchesSemiMonthlyPaychecks(dates, first, second) {
				candidates = append(candidates, semiMonthlyPaycheckResult(dates[len(dates)-1], first, second))
			}
		}
	}
	if len(candidates) != 1 {
		return paycheckCalendar{}, false
	}
	return candidates[0], true
}

func matchesSemiMonthlyPaychecks(dates []time.Time, first, second int) bool {
	index := 0
	last := dates[len(dates)-1]
	for month := paycheckMonth(dates[0]); !month.After(last); month = paycheckNextMonth(month) {
		for _, day := range []int{first, second} {
			date := paycheckCalendarDay(month, day)
			if date.Before(dates[0]) || date.After(last) {
				continue
			}
			if index >= len(dates) || !date.Equal(dates[index]) {
				return false
			}
			index++
		}
	}
	return index == len(dates)
}

func semiMonthlyPaycheckResult(last time.Time, first, second int) paycheckCalendar {
	schedule := paycheckSchedule(last)
	schedule.SemiMonthlyDays = []budget.DayOfMonth{budget.DayOfMonth(first), budget.DayOfMonth(second)}
	next := paycheckCalendarDay(last, second)
	if !next.After(last) {
		next = paycheckCalendarDay(paycheckNextMonth(last), first)
	}
	return paycheckCalendar{frequency: "semimonthly", schedule: schedule, next: next}
}

func paycheckMonth(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func paycheckMonthNumber(date time.Time) int {
	return date.Year()*12 + int(date.Month())
}

func paycheckNextMonth(date time.Time) time.Time {
	return paycheckMonth(date).AddDate(0, 1, 0)
}

func paycheckCalendarDay(month time.Time, day int) time.Time {
	start := paycheckMonth(month)
	lastDay := start.AddDate(0, 1, -1).Day()
	return start.AddDate(0, 0, min(day, lastDay)-1)
}
