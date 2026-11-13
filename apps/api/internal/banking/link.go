package banking

import (
	"context"
	"errors"
	"time"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

type attempt struct {
	ID, ConnectionID, HouseholdID, MemberID, SessionHash, Kind string
	Update                                                     bool
	ExpiresAt                                                  time.Time
}

// CreateLink binds a short-lived Link token to one verified session and connecting member.
// A production budget slot is reserved before Link can create an Item and is never refunded.
func (s *Service) CreateLink(ctx context.Context, identity Identity, request budget.LinkTokenRequest) (budget.LinkTokenResponse, error) {
	if err := validateLinkRequest(request); err != nil {
		return budget.LinkTokenResponse{}, err
	}
	if err := s.acquire(ctx); err != nil {
		return budget.LinkTokenResponse{}, err
	}
	defer s.release()
	pending, token, err := s.reserveAttempt(ctx, identity, request)
	if err != nil {
		return budget.LinkTokenResponse{}, err
	}
	link, expires, err := s.provider.Link(ctx, identity.HouseholdID+"/"+identity.MemberID, pending.Kind, token)
	if err != nil {
		return budget.LinkTokenResponse{}, s.failLink(pending, err)
	}
	cipher, err := s.vault.Seal(link, credentialContext(identity.HouseholdID, pending.ID, "link"))
	if err != nil {
		return budget.LinkTokenResponse{}, s.failLink(pending, err)
	}
	if expires.After(pending.ExpiresAt) {
		expires = pending.ExpiresAt
	}
	tag, err := s.db.Pool.Exec(ctx, `UPDATE bank_link_attempts SET token_cipher=$1,expires_at=$2 WHERE id=$3 AND state='pending'`, cipher, expires, pending.ID)
	if err != nil {
		return budget.LinkTokenResponse{}, s.failLink(pending, err)
	}
	if tag.RowsAffected() != 1 {
		return budget.LinkTokenResponse{}, ErrConflict
	}
	return budget.LinkTokenResponse{ConnectionID: pending.ConnectionID, AttemptID: pending.ID, LinkToken: link, ExpiresAt: expires.UTC().Format(time.RFC3339)}, nil
}

func (s *Service) reserveAttempt(ctx context.Context, identity Identity, request budget.LinkTokenRequest) (attempt, string, error) {
	id, err := opaqueID()
	if err != nil {
		return attempt{}, "", err
	}
	pending := attempt{ID: id, HouseholdID: identity.HouseholdID, MemberID: identity.MemberID, SessionHash: identity.SessionHash, ExpiresAt: time.Now().Add(AttemptLifetime)}
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return pending, "", err
	}
	defer rollback(tx)
	var household string
	if err := tx.QueryRow(ctx, `SELECT id FROM households WHERE id=$1 FOR UPDATE`, identity.HouseholdID).Scan(&household); err != nil {
		return pending, "", err
	}
	if err := retireIssuedAttempt(ctx, tx, identity, s.config.Environment); err != nil {
		return pending, "", err
	}
	token, err := s.prepareConnection(ctx, tx, identity, request, &pending)
	if err != nil {
		return pending, "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO bank_link_attempts(id,household_id,member_id,session_hash,connection_id,is_update,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, identity.HouseholdID, identity.MemberID, identity.SessionHash, pending.ConnectionID, pending.Update, pending.ExpiresAt)
	if err != nil {
		return pending, "", err
	}
	return pending, token, tx.Commit(ctx)
}

func (s *Service) prepareConnection(ctx context.Context, tx pgx.Tx, identity Identity, request budget.LinkTokenRequest, pending *attempt) (string, error) {
	if request.ConnectionID != nil {
		if request.Kind != nil {
			return "", ErrInvalid
		}
		pending.Update = true
		pending.ConnectionID = *request.ConnectionID
		var cipher string
		err := tx.QueryRow(ctx, `SELECT kind,token_cipher FROM bank_connections WHERE id=$1 AND household_id=$2 AND member_id=$3 AND state NOT IN ('pending','disconnecting','disconnected') AND environment=$4`, pending.ConnectionID, identity.HouseholdID, identity.MemberID, s.config.Environment).Scan(&pending.Kind, &cipher)
		if err != nil {
			return "", notFound(err)
		}
		return s.vault.Open(cipher, credentialContext(identity.HouseholdID, pending.ConnectionID, "access"))
	}
	if request.Kind == nil || (*request.Kind != "depository" && *request.Kind != "investment") {
		return "", ErrInvalid
	}
	pending.Kind = string(*request.Kind)
	id, err := opaqueID()
	if err != nil {
		return "", err
	}
	pending.ConnectionID = id
	if err := s.reserveBudget(ctx, tx); err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO bank_connections(id,household_id,member_id,kind,environment) VALUES($1,$2,$3,$4,$5)`, id, identity.HouseholdID, identity.MemberID, pending.Kind, s.config.Environment)
	return "", err
}

func (s *Service) reserveBudget(ctx context.Context, tx pgx.Tx) error {
	if s.config.Environment != "production" {
		return nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO bank_item_budget(environment) VALUES('production') ON CONFLICT DO NOTHING`)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE bank_item_budget SET reservations=reservations+1 WHERE environment='production' AND reservations<$1`, s.config.LifetimeItemLimit)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrLimit
	}
	return nil
}

// PendingLink restores OAuth only for the same live session; it exposes no access token.
func (s *Service) PendingLink(ctx context.Context, identity Identity) (budget.LinkTokenResponse, error) {
	if !s.config.Enabled {
		return budget.LinkTokenResponse{}, ErrUnavailable
	}
	var result budget.LinkTokenResponse
	var encrypted string
	var expires time.Time
	err := s.db.Pool.QueryRow(ctx, `SELECT id,connection_id,token_cipher,expires_at FROM bank_link_attempts WHERE household_id=$1 AND member_id=$2 AND session_hash=$3 AND state='pending' AND expires_at>now() AND token_cipher IS NOT NULL AND EXISTS(SELECT 1 FROM bank_connections c WHERE c.id=connection_id AND c.environment=$4) ORDER BY created_at DESC LIMIT 1`, identity.HouseholdID, identity.MemberID, identity.SessionHash, s.config.Environment).Scan(&result.AttemptID, &result.ConnectionID, &encrypted, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrNotFound
	}
	if err != nil {
		return result, err
	}
	result.LinkToken, err = s.vault.Open(encrypted, credentialContext(identity.HouseholdID, result.AttemptID, "link"))
	result.ExpiresAt = expires.UTC().Format(time.RFC3339)
	return result, err
}
