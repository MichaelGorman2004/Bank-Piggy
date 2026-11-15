package activity

import (
	"errors"
	"strings"
	"time"

	"bankpiggy/api/internal/budget"
)

// ErrInvalid rejects unsupported scopes and values instead of producing misleading totals.
var ErrInvalid = errors.New("activity cannot be projected safely")

// Project derives monthly actuals only from selected accounts and separately stored history.
// Reconciliation precedes month/scope filtering, and never changes the supplied plan or records.
func Project(plan budget.Household, records []budget.ActivityRecord, sources []budget.ActivitySource, month time.Time, scope string, now time.Time) (budget.ActivityResponse, error) {
	month = time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	result := emptyResponse(plan, month, scope, sources)
	if scope != "household" && !budget.HasMember(plan, scope) {
		return result, ErrInvalid
	}
	included := IncludedAccounts(sources)
	reconciled := Reconcile(ownedActivity(records, included))
	for _, suggestion := range Suggestions(reconciled, included, now) {
		if inScope(suggestion.Ownership, scope) {
			result.Suggestions = append(result.Suggestions, suggestion)
		}
	}
	monthly, unresolved, err := monthlyRecords(reconciled, included, plan, result.Month, scope, now)
	if err != nil {
		return result, err
	}
	if err := totalRecords(&result.Totals, monthly); err != nil {
		return result, err
	}
	if err := compareMonth(&result, plan, month); err != nil {
		return result, err
	}
	result.Coverage, result.Notice = coverage(sources, month, "household", now, unresolved || unsupportedCurrency(monthly))
	return result, nil
}

func monthlyRecords(records []budget.ActivityRecord, included map[string]budget.ActivityAccount, plan budget.Household, month, scope string, now time.Time) ([]budget.ActivityRecord, bool, error) {
	result := []budget.ActivityRecord{}
	unresolved := false
	for _, record := range records {
		account, exists := included[record.AccountID]
		if !exists || !strings.HasPrefix(string(record.Date), month+"-") {
			continue
		}
		record.Ownership = account.Ownership
		var err error
		record, err = allocateRecord(record, scope, plan)
		if err != nil {
			return nil, false, err
		}
		if !inScope(recordOwnership(record), scope) {
			continue
		}
		if string(record.Date) > now.UTC().Format(time.DateOnly) {
			record = reason(record, "excluded", "The posting date is in the future. This activity is not included yet.")
		}
		result = append(result, record)
		if record.Category == "review" {
			unresolved = true
		}
	}
	return result, unresolved, nil
}

func emptyResponse(plan budget.Household, month time.Time, scope string, sources []budget.ActivitySource) budget.ActivityResponse {
	return budget.ActivityResponse{Month: month.Format("2006-01"), Scope: scope, PlanRevision: plan.Revision,
		Sources: sources, Suggestions: []budget.PaycheckSuggestion{}}
}

// IncludedAccounts selects available accounts only after source consent and operator approval.
func IncludedAccounts(sources []budget.ActivitySource) map[string]budget.ActivityAccount {
	result := map[string]budget.ActivityAccount{}
	for _, source := range sources {
		if !source.Enabled || !source.Allowed || source.NeedsConsent || source.State == "unsupported" || source.State == "disabled" {
			continue
		}
		for _, account := range source.Accounts {
			if account.Enabled && account.Available {
				result[account.AccountID] = account
			}
		}
	}
	return result
}

func ownedActivity(records []budget.ActivityRecord, accounts map[string]budget.ActivityAccount) []budget.ActivityRecord {
	owned := append([]budget.ActivityRecord(nil), records...)
	for index := range owned {
		if account, ok := accounts[owned[index].AccountID]; ok {
			owned[index].Ownership = account.Ownership
		}
	}
	return owned
}

func inScope(owner, scope string) bool {
	return scope == "household" || owner == "shared" || owner == scope
}

func scopedCents(cents int64, ownership, scope string, plan budget.Household) int64 {
	if scope == "household" || ownership != "shared" || len(plan.Members) == 1 {
		return cents
	}
	if scope == plan.Members[0].ID {
		return cents - cents/2
	}
	return cents / 2
}

func coverage(sources []budget.ActivitySource, month time.Time, scope string, now time.Time, unresolved bool) (string, string) {
	if len(sources) == 0 {
		return "not_connected", "Connect accounts to start recording monthly activity."
	}
	selected, complete := 0, true
	for _, source := range sources {
		if sourceSelected(source, scope) {
			selected++
			complete = complete && sourceCovers(source, month, now) && selectedAccountsAvailable(source, scope)
		}
	}
	if selected == 0 {
		return "disabled", "Enable history for the accounts you want in monthly totals."
	}
	if !complete || unresolved {
		return "incomplete", "Recorded activity is incomplete. Missing history, unsupported values, and activity with an uncertain type are not included."
	}
	return "ready", "Posted activity from selected accounts. Coverage uses all selected household accounts. Existing saved shares remain fixed; other activity follows account ownership. Other accounts are not included."
}

func unsupportedCurrency(records []budget.ActivityRecord) bool {
	for _, record := range records {
		if !record.Pending && !record.Cancelled && (record.Currency == nil || *record.Currency != "USD") {
			return true
		}
	}
	return false
}

func sourceSelected(source budget.ActivitySource, scope string) bool {
	if !source.Enabled {
		return false
	}
	for _, account := range source.Accounts {
		if account.Enabled && inScope(account.Ownership, scope) {
			return true
		}
	}
	return false
}

func selectedAccountsAvailable(source budget.ActivitySource, scope string) bool {
	for _, account := range source.Accounts {
		if account.Enabled && inScope(account.Ownership, scope) && !account.Available {
			return false
		}
	}
	return true
}

func sourceCovers(source budget.ActivitySource, month, now time.Time) bool {
	if !source.Allowed || source.NeedsConsent || source.State != "ready" || source.HistoryStart == nil || source.HistoryEnd == nil {
		return false
	}
	end := month.AddDate(0, 1, -1).Format("2006-01-02")
	if today := now.UTC().Format("2006-01-02"); end > today {
		end = today
	}
	return month.Format("2006-01-02") <= end && *source.HistoryStart <= month.Format("2006-01-02") && *source.HistoryEnd >= end
}
