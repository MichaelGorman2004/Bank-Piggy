package banking

import (
	"context"
	"encoding/json"
	"time"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

// List returns household-scoped snapshots with a separate effective balance projection.
func (s *Service) List(ctx context.Context, identity Identity) (budget.ConnectionList, error) {
	result := budget.ConnectionList{Enabled: s.config.Enabled, Environment: s.config.Environment, Connections: []budget.Connection{}}
	if !s.config.Enabled {
		result.Environment = "disabled"
	}
	tx, err := s.db.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return result, err
	}
	defer rollback(tx)
	var data []byte
	if err := tx.QueryRow(ctx, "SELECT plan FROM households WHERE id=$1", identity.HouseholdID).Scan(&data); err != nil {
		return result, err
	}
	var plan budget.Household
	if err := json.Unmarshal(data, &plan); err != nil {
		return result, err
	}
	result.PlanRevision = plan.Revision
	connections := []budget.Connection{}
	if s.config.Enabled {
		connections, err = readConnections(ctx, tx, identity, s.config.Environment)
	}
	if err != nil {
		return result, err
	}
	result.Connections = connections
	result.EffectiveBalances = EffectiveBalances(plan, connections)
	return result, tx.Commit(ctx)
}

func readConnections(ctx context.Context, tx pgx.Tx, identity Identity, environment string) ([]budget.Connection, error) {
	rows, err := tx.Query(ctx, `SELECT c.id,c.institution_name,c.member_id,c.kind,c.state,c.last_synced_at,c.next_sync_at,c.error_code,c.failures,
 (SELECT a.id FROM bank_link_attempts a WHERE a.connection_id=c.id AND a.state='completed' ORDER BY a.created_at DESC,a.id DESC LIMIT 1)
 FROM bank_connections c WHERE c.household_id=$1 AND c.environment=$2 ORDER BY c.created_at,c.id`, identity.HouseholdID, environment)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []budget.Connection{}
	for rows.Next() {
		var c budget.Connection
		var last, next *time.Time
		var failures int
		if err := rows.Scan(&c.ID, &c.InstitutionName, &c.ConnectedByMemberID, &c.Kind, &c.State, &last, &next, &c.ErrorCode, &failures, &c.LastCompletedAttemptID); err != nil {
			return nil, err
		}
		decorateConnection(&c, identity, last, next, failures)
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for index := range result {
		accounts, err := readAccounts(ctx, tx, result[index].ID)
		if err != nil {
			return nil, err
		}
		result[index].Accounts = accounts
	}
	return result, nil
}

func readAccounts(ctx context.Context, tx pgx.Tx, id string) ([]budget.ConnectedAccount, error) {
	rows, err := tx.Query(ctx, `SELECT id,snapshot,active,mapping,mapping_revision FROM bank_accounts WHERE connection_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []budget.ConnectedAccount{}
	for rows.Next() {
		var account budget.ConnectedAccount
		var snapshot, mapping []byte
		if err := rows.Scan(&account.ID, &snapshot, &account.Active, &mapping, &account.MappingRevision); err != nil {
			return nil, err
		}
		savedID, active, revision := account.ID, account.Active, account.MappingRevision
		if err := json.Unmarshal(snapshot, &account); err != nil {
			return nil, err
		}
		account.ID, account.Active, account.MappingRevision = savedID, active, revision
		if err := json.Unmarshal(mapping, &account.Mapping); err != nil {
			return nil, err
		}
		result = append(result, account)
	}
	return result, rows.Err()
}

func scheduledState(state budget.ConnectionState, failures int) bool {
	if failures >= MaxFailures {
		return false
	}
	return state == "connected" || state == "delayed" || state == "disconnecting"
}

func decorateConnection(c *budget.Connection, identity Identity, last, next *time.Time, failures int) {
	c.LastSyncedAt = formatTime(last)
	if scheduledState(c.State, failures) {
		c.NextSyncAt = formatTime(next)
	}
	c.CanManage = c.ConnectedByMemberID == identity.MemberID && c.State != "disconnected"
	c.CanReconnect = c.CanManage && c.State != "disconnecting" && c.State != "pending"
	c.Accounts = []budget.ConnectedAccount{}
}
