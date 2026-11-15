package banking

import (
	"context"
	"encoding/json"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

// SetActivitySettings saves explicit source consent and account scope. Disabling
// invalidates in-flight work but does not cancel the provider's subscription.
func (s *Service) SetActivitySettings(ctx context.Context, identity Identity, connectionID string, request budget.ActivitySettingsRequest) (budget.ActivitySource, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return budget.ActivitySource{}, err
	}
	defer rollback(tx)
	plan, source, err := s.activitySettingsInputs(ctx, tx, identity, connectionID)
	if err != nil {
		return source, err
	}
	if source.Revision != request.ExpectedRevision {
		return source, ErrConflict
	}
	if request.Enabled && !source.Allowed {
		return source, ErrUnavailable
	}
	if err := validateActivitySettings(plan, source, request); err != nil {
		return source, err
	}
	if err := persistActivitySettings(ctx, tx, identity.HouseholdID, source, request); err != nil {
		return source, err
	}
	source, err = s.readActivitySource(ctx, tx, identity, connectionID)
	if err != nil {
		return source, err
	}
	return source, tx.Commit(ctx)
}

func persistActivitySettings(ctx context.Context, tx pgx.Tx, household string, source budget.ActivitySource, request budget.ActivitySettingsRequest) error {
	data, err := json.Marshal(request.Accounts)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO bank_activity_jobs(connection_id,household_id,enabled,accounts,authorized_at)
 VALUES($1,$2,$3,$4,CASE WHEN $5='investment' AND $3 THEN now() ELSE NULL END)
 ON CONFLICT(connection_id) DO UPDATE SET revision=bank_activity_jobs.revision+1,
 enabled=EXCLUDED.enabled,accounts=EXCLUDED.accounts,generation=bank_activity_jobs.generation+1,
 authorized_at=COALESCE(bank_activity_jobs.authorized_at,EXCLUDED.authorized_at),
 failures=0,error_code=NULL,lease_owner=NULL,lease_until=NULL`, source.ConnectionID, household, request.Enabled, data, string(source.Kind))
	return err
}

func (s *Service) activitySettingsInputs(ctx context.Context, tx pgx.Tx, identity Identity, id string) (budget.Household, budget.ActivitySource, error) {
	var plan budget.Household
	var data []byte
	var source budget.ActivitySource
	if err := tx.QueryRow(ctx, `SELECT plan FROM households WHERE id=$1 FOR UPDATE`, identity.HouseholdID).Scan(&data); err != nil {
		return plan, source, err
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		return plan, source, err
	}
	var owner string
	err := tx.QueryRow(ctx, `SELECT member_id FROM bank_connections WHERE id=$1 AND household_id=$2 AND environment=$3 AND state NOT IN ('pending','disconnecting','disconnected') FOR UPDATE`, id, identity.HouseholdID, s.config.Environment).Scan(&owner)
	if err != nil {
		return plan, source, notFound(err)
	}
	if owner != identity.MemberID {
		return plan, source, ErrNotFound
	}
	source, err = s.readActivitySource(ctx, tx, identity, id)
	return plan, source, err
}

func (s *Service) readActivitySource(ctx context.Context, tx pgx.Tx, identity Identity, id string) (budget.ActivitySource, error) {
	sources, err := s.readActivitySources(ctx, tx, identity)
	if err != nil {
		return budget.ActivitySource{}, err
	}
	for _, source := range sources {
		if source.ConnectionID == id {
			return source, nil
		}
	}
	return budget.ActivitySource{}, ErrNotFound
}

func validateActivitySettings(plan budget.Household, source budget.ActivitySource, request budget.ActivitySettingsRequest) error {
	if request.ExpectedRevision < 0 || request.Accounts == nil || len(request.Accounts) > maxProviderAccounts {
		return ErrInvalid
	}
	available := map[string]bool{}
	for _, account := range source.Accounts {
		available[account.AccountID] = account.Available || account.Enabled
	}
	seen := map[string]bool{}
	for _, account := range request.Accounts {
		allowed, exists := available[account.AccountID]
		if !exists || seen[account.AccountID] {
			return ErrInvalid
		}
		if err := validateActivitySelection(plan, account, allowed); err != nil {
			return err
		}
		seen[account.AccountID] = true
	}
	return nil
}

func validateActivitySelection(plan budget.Household, account budget.ActivityAccountSelection, allowed bool) error {
	if account.Enabled && !allowed {
		return ErrInvalid
	}
	if account.Ownership != "shared" && !budget.HasMember(plan, account.Ownership) {
		return ErrInvalid
	}
	return nil
}
