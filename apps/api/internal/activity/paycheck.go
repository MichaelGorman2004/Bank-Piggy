package activity

import (
	"sort"
	"strings"
	"time"

	"bankpiggy/api/internal/budget"
)

const paycheckHistoryDays = 180
const paycheckPostingGraceDays = 3
const paycheckDateLayout = "2006-01-02"

type paycheckStream struct {
	account budget.ActivityAccount
	payer   string
	records []budget.ActivityRecord
}

// Suggestions estimates regular net deposits from selected, posted USD bank activity.
// It does not infer a wage earner for shared accounts or change any saved plan fields.
func Suggestions(records []budget.ActivityRecord, included map[string]budget.ActivityAccount, now time.Time) []budget.PaycheckSuggestion {
	today := paycheckToday(now)
	groups := paycheckGroups(records, included, today)
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := []budget.PaycheckSuggestion{}
	for _, key := range keys {
		if suggestion, ok := suggestPaycheck(groups[key], today); ok {
			result = append(result, suggestion)
		}
	}
	return result
}

func paycheckToday(now time.Time) time.Time {
	utc := now.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func paycheckGroups(records []budget.ActivityRecord, included map[string]budget.ActivityAccount, today time.Time) map[string]paycheckStream {
	groups := map[string]paycheckStream{}
	for _, record := range records {
		account, exists := included[record.AccountID]
		if !exists || !eligiblePaycheck(record, account, today) {
			continue
		}
		payer := normalizedPaycheckPayer(record)
		if payer == "" {
			continue
		}
		key := record.AccountID + "\x00" + payer
		stream := groups[key]
		stream.account, stream.payer = account, payer
		stream.records = append(stream.records, record)
		groups[key] = stream
	}
	return groups
}

func eligiblePaycheck(record budget.ActivityRecord, account budget.ActivityAccount, today time.Time) bool {
	if !eligiblePaycheckAccount(account) || !postedPaycheck(record) || record.AmountCents == nil {
		return false
	}
	if *record.AmountCents >= 0 || *record.AmountCents < -budget.MaxMoneyCents || !budget.ValidDate(string(record.Date)) {
		return false
	}
	date, err := time.Parse(paycheckDateLayout, string(record.Date))
	return err == nil && !date.After(today) && !date.Before(today.AddDate(0, 0, -paycheckHistoryDays))
}

func eligiblePaycheckAccount(account budget.ActivityAccount) bool {
	return account.Enabled && account.Available && account.Currency != nil && *account.Currency == "USD"
}

func postedPaycheck(record budget.ActivityRecord) bool {
	return record.Kind == "bank" && record.Category == "paycheck" && !record.Pending && !record.Cancelled &&
		record.Currency != nil && *record.Currency == "USD"
}

func normalizedPaycheckPayer(record budget.ActivityRecord) string {
	payer := record.Counterparty
	if strings.TrimSpace(payer) == "" {
		payer = record.Name
	}
	return strings.ToLower(strings.Join(strings.Fields(payer), " "))
}

func suggestPaycheck(stream paycheckStream, today time.Time) (budget.PaycheckSuggestion, bool) {
	sort.Slice(stream.records, func(i, j int) bool {
		if stream.records[i].Date == stream.records[j].Date {
			return stream.records[i].ID < stream.records[j].ID
		}
		return stream.records[i].Date < stream.records[j].Date
	})
	dates := paycheckDates(stream.records)
	calendar, ok := inferPaycheckCalendar(dates)
	if !ok || calendar.next.Before(today.AddDate(0, 0, -paycheckPostingGraceDays)) {
		return budget.PaycheckSuggestion{}, false
	}
	amounts, typical := paycheckAmounts(stream.records)
	monthly, ok := paycheckMonthlyAmount(typical, calendar.frequency)
	if !ok {
		return budget.PaycheckSuggestion{}, false
	}
	suggestion := paycheckResult(stream, calendar, typical, monthly, amounts)
	id, err := paycheckFingerprint(stream, suggestion)
	suggestion.ID = id
	return suggestion, err == nil
}

func paycheckDates(records []budget.ActivityRecord) []time.Time {
	dates := make([]time.Time, 0, len(records))
	for _, record := range records {
		date, err := time.Parse(paycheckDateLayout, string(record.Date))
		if err != nil {
			return nil
		}
		dates = append(dates, date)
	}
	return dates
}

func paycheckResult(stream paycheckStream, calendar paycheckCalendar, typical, monthly int64, amounts []int64) budget.PaycheckSuggestion {
	result := budget.PaycheckSuggestion{Source: stream.account.Name + " · " + stream.payer,
		Ownership: stream.account.Ownership, DepositCount: int64(len(stream.records)),
		LastPayDate: stream.records[len(stream.records)-1].Date, TypicalPaycheckCents: typical,
		MonthlyTakeHomeCents: monthly, Frequency: calendar.frequency, PaySchedule: calendar.schedule,
		VariableAmount: paycheckVariableAmount(amounts, typical), Deposits: []budget.PaycheckDeposit{}}
	if stream.account.Ownership != "shared" {
		owner := stream.account.Ownership
		result.MemberID = &owner
	}
	if next := calendar.next.Format(paycheckDateLayout); budget.ValidDate(next) {
		date := budget.CalendarDate(next)
		result.NextPayDate = &date
	}
	for _, record := range stream.records {
		result.Deposits = append(result.Deposits, budget.PaycheckDeposit{Date: record.Date, AmountCents: -*record.AmountCents})
	}
	return result
}
