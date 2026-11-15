package banking

import (
	"context"
	"encoding/json"
	"time"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

type activitySourceRow struct {
	source                 budget.ActivitySource
	owner, connectionState string
	authorized, complete   bool
	selections             []byte
	last, start, end       *time.Time
	failures               int
}

func (s *Service) activityAllowed(kind string) bool {
	if !s.config.Enabled {
		return false
	}
	if s.config.Environment == "production" && !s.config.ProductionApproved {
		return false
	}
	if kind == "investment" {
		return s.config.InvestmentActivityApproved
	}
	return s.config.BankActivityApproved
}

// readActivitySources reads only saved snapshots. It never starts provider work.
func (s *Service) readActivitySources(ctx context.Context, tx pgx.Tx, identity Identity) ([]budget.ActivitySource, error) {
	rows, err := tx.Query(ctx, `SELECT c.id,c.institution_name,c.kind,c.member_id,c.state,
 COALESCE(j.revision,0),COALESCE(j.enabled,false),j.authorized_at IS NOT NULL,
 COALESCE(j.accounts,'[]'),COALESCE(j.history_complete,false),j.last_synced_at,
 j.history_start,j.history_end,j.error_code,COALESCE(j.failures,0)
 FROM bank_connections c LEFT JOIN bank_activity_jobs j ON j.connection_id=c.id
 WHERE c.household_id=$1 AND c.environment=$2 AND c.state NOT IN ('pending','disconnected')
 ORDER BY c.created_at,c.id`, identity.HouseholdID, s.config.Environment)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []activitySourceRow{}
	for rows.Next() {
		var value activitySourceRow
		v := &value.source
		if err := rows.Scan(&v.ConnectionID, &v.InstitutionName, &v.Kind, &value.owner, &value.connectionState, &v.Revision, &v.Enabled, &value.authorized, &value.selections, &value.complete, &value.last, &value.start, &value.end, &v.ErrorCode, &value.failures); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	result := make([]budget.ActivitySource, 0, len(values))
	for _, value := range values {
		accounts, err := readAccounts(ctx, tx, value.source.ConnectionID, value.owner)
		if err != nil {
			return nil, err
		}
		if err := s.decorateActivitySource(&value, accounts, identity); err != nil {
			return nil, err
		}
		result = append(result, value.source)
	}
	return result, nil
}

func (s *Service) decorateActivitySource(value *activitySourceRow, accounts []budget.ConnectedAccount, identity Identity) error {
	var saved []budget.ActivityAccountSelection
	if err := json.Unmarshal(value.selections, &saved); err != nil {
		return err
	}
	source := &value.source
	source.Allowed = s.activityAllowed(string(source.Kind))
	source.CanManage = value.owner == identity.MemberID && value.connectionState != "disconnecting"
	source.NeedsConsent = source.Enabled && !value.authorized && source.Kind != "investment"
	source.LastSyncedAt = formatTime(value.last)
	if value.complete {
		source.HistoryStart, source.HistoryEnd = activityDate(value.start), activityDate(value.end)
	}
	source.State = activitySourceState(*value)
	source.Accounts = make([]budget.ActivityAccount, 0, len(accounts))
	seen := map[string]bool{}
	for _, account := range accounts {
		source.Accounts = append(source.Accounts, activityAccount(account, value.owner, string(source.Kind), source.Revision, saved))
		seen[account.ID] = true
	}
	for _, selection := range saved {
		if selection.Enabled && !seen[selection.AccountID] {
			source.Accounts = append(source.Accounts, budget.ActivityAccount{AccountID: selection.AccountID, Name: "Unavailable account", Enabled: true, Ownership: selection.Ownership})
		}
	}
	return nil
}

func activityAccount(account budget.ConnectedAccount, owner, kind string, revision int64, saved []budget.ActivityAccountSelection) budget.ActivityAccount {
	available := activityAccountAvailable(account, kind)
	value := budget.ActivityAccount{AccountID: account.ID, Name: account.Name, ProviderType: account.ProviderType, Currency: account.Currency, Available: available, Ownership: owner}
	if revision == 0 {
		value.Enabled = available && (account.MappingRevision <= 1 || account.Mapping.Enabled)
		if account.MappingRevision > 1 && account.Mapping.Ownership != "" {
			value.Ownership = string(account.Mapping.Ownership)
		}
		return value
	}
	for _, selection := range saved {
		if selection.AccountID == account.ID {
			value.Enabled, value.Ownership = selection.Enabled, selection.Ownership
		}
	}
	return value
}

func activityAccountAvailable(account budget.ConnectedAccount, kind string) bool {
	if !account.Active || account.Currency == nil || *account.Currency != "USD" {
		return false
	}
	if kind == "investment" {
		return account.ProviderType == "investment"
	}
	return account.ProviderType == "depository" || account.ProviderType == "credit"
}

func activitySourceState(value activitySourceRow) string {
	source := value.source
	switch {
	case !source.Enabled:
		return "disabled"
	case !source.Allowed:
		return "disabled"
	case value.connectionState == "disconnecting":
		return "disabled"
	case value.connectionState == "reconnect":
		return "reconnect"
	case source.NeedsConsent:
		return "consent_required"
	case source.ErrorCode != nil || value.failures > 0:
		return "delayed"
	case !value.complete:
		return "initializing"
	default:
		return "ready"
	}
}

func activityDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	return ptr(value.Format(time.DateOnly))
}
