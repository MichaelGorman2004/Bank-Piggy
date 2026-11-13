package banking

import (
	"context"
	"encoding/json"
	"errors"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MapAccount validates explicit ownership and manual reconciliation under the plan lock.
func (s *Service) MapAccount(ctx context.Context, identity Identity, connectionID, accountID string, request budget.AccountMappingRequest) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	plan, account, err := mappingInputs(ctx, tx, identity.HouseholdID, connectionID, accountID, s.config.Environment)
	if err != nil {
		return err
	}
	if account.MappingRevision != request.Revision {
		return ErrConflict
	}
	if err := validateMapping(plan, account, request); err != nil {
		return err
	}
	if err := persistMapping(ctx, tx, accountID, request); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func mappingInputs(ctx context.Context, tx pgx.Tx, household, connection, id, environment string) (budget.Household, budget.ConnectedAccount, error) {
	var plan budget.Household
	var account budget.ConnectedAccount
	var planData, snapshot []byte
	if err := tx.QueryRow(ctx, `SELECT plan FROM households WHERE id=$1 FOR UPDATE`, household).Scan(&planData); err != nil {
		return plan, account, err
	}
	if err := json.Unmarshal(planData, &plan); err != nil {
		return plan, account, err
	}
	var state string
	if err := tx.QueryRow(ctx, `SELECT state FROM bank_connections WHERE id=$1 AND household_id=$2 AND state NOT IN ('disconnecting','disconnected') AND environment=$3 FOR UPDATE`, connection, household, environment).Scan(&state); err != nil {
		return plan, account, notFound(err)
	}
	var revision int64
	var active bool
	if err := tx.QueryRow(ctx, `SELECT snapshot,mapping_revision,active FROM bank_accounts WHERE connection_id=$1 AND id=$2 FOR UPDATE`, connection, id).Scan(&snapshot, &revision, &active); err != nil {
		return plan, account, notFound(err)
	}
	if err := json.Unmarshal(snapshot, &account); err != nil {
		return plan, account, err
	}
	account.ID, account.MappingRevision, account.Active = id, revision, active
	return plan, account, nil
}

func persistMapping(ctx context.Context, tx pgx.Tx, id string, request budget.AccountMappingRequest) error {
	mapping := budget.AccountMapping{Enabled: request.Enabled, Ownership: request.Ownership, Classification: request.Classification, ManualMemberID: request.ManualMemberID, ManualAccountID: request.ManualAccountID}
	data, err := json.Marshal(mapping)
	if err != nil {
		return err
	}
	member, manual := mapping.ManualMemberID, mapping.ManualAccountID
	if !mapping.Enabled {
		member, manual = nil, nil
	}
	_, err = tx.Exec(ctx, `UPDATE bank_accounts SET mapping=$1,mapping_revision=mapping_revision+1,manual_member_id=$2,manual_account_id=$3 WHERE id=$4`, data, member, manual, id)
	var constraint *pgconn.PgError
	if errors.As(err, &constraint) && constraint.Code == "23505" {
		return ErrConflict
	}
	return err
}

func validateMapping(plan budget.Household, account budget.ConnectedAccount, request budget.AccountMappingRequest) error {
	if request.Revision < 1 || !validClassification(request.Classification) {
		return ErrInvalid
	}
	if request.Ownership != "shared" && !budget.HasMember(plan, string(request.Ownership)) {
		return ErrInvalid
	}
	if request.Enabled && (!account.Active || !supportedAccount(account) || request.Classification == "unclassified") {
		return ErrInvalid
	}
	return validateManualMapping(plan, request)
}

func validateManualMapping(plan budget.Household, request budget.AccountMappingRequest) error {
	if (request.ManualMemberID == nil) != (request.ManualAccountID == nil) {
		return ErrInvalid
	}
	if request.ManualMemberID != nil && !hasManualAccount(plan, *request.ManualMemberID, *request.ManualAccountID) {
		return ErrInvalid
	}
	return nil
}

func validClassification(kind budget.ConnectedAccountType) bool {
	switch kind {
	case "unclassified", "checking", "hsa", "traditional_401k", "roth_401k", "traditional_ira", "roth_ira", "brokerage", "high_yield_savings", "savings", "other_investment":
		return true
	default:
		return false
	}
}
func supportedAccount(account budget.ConnectedAccount) bool {
	return account.ProviderType == "depository" || account.ProviderType == "investment"
}
func hasManualAccount(plan budget.Household, memberID, accountID string) bool {
	for _, member := range plan.Members {
		if member.ID == memberID {
			for _, account := range member.Accounts {
				if account.ID == accountID {
					return true
				}
			}
		}
	}
	return false
}
