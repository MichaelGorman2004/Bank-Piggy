package banking

import (
	"context"
	"time"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
	plaid "github.com/plaid/plaid-go/v50/legacy"
)

const activityHistoryDays = 180

func (s *Service) createProviderLink(ctx context.Context, identity Identity, pending attempt, token string) (string, time.Time, error) {
	user := identity.HouseholdID + "/" + identity.MemberID
	if pending.Purpose == "activity" {
		return s.provider.ActivityLink(ctx, user, token)
	}
	return s.provider.Link(ctx, user, pending.Kind, token)
}

func activityPurpose(purpose string) *budget.LinkPurpose {
	if purpose == "activity" {
		return ptr(budget.LinkPurpose(purpose))
	}
	return nil
}

func (s *Service) prepareActivityConsent(ctx context.Context, tx pgx.Tx, pending *attempt) error {
	if pending.Kind != "depository" || !s.activityAllowed(pending.Kind) {
		return ErrUnavailable
	}
	var revision int64
	err := tx.QueryRow(ctx, `SELECT revision FROM bank_activity_jobs WHERE connection_id=$1 AND enabled FOR UPDATE`, pending.ConnectionID).Scan(&revision)
	if err != nil {
		return notFound(err)
	}
	pending.ActivityRevision = &revision
	return nil
}

func (s *Service) finishActivityConsent(ctx context.Context, pending attempt) error {
	if !s.activityAllowed(pending.Kind) || pending.ActivityRevision == nil {
		return ErrUnavailable
	}
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err := lockBankHousehold(ctx, tx, pending.HouseholdID); err != nil {
		return err
	}
	var id string
	err = tx.QueryRow(ctx, `SELECT id FROM bank_connections WHERE id=$1 AND household_id=$2 AND member_id=$3 AND environment=$4 AND state NOT IN ('disconnecting','disconnected') FOR UPDATE`, pending.ConnectionID, pending.HouseholdID, pending.MemberID, s.config.Environment).Scan(&id)
	if err != nil {
		return notFound(err)
	}
	tag, err := tx.Exec(ctx, `UPDATE bank_activity_jobs SET authorized_at=now(),generation=generation+1,
 next_sync_at=now(),failures=0,error_code=NULL,lease_owner=NULL,lease_until=NULL
 WHERE connection_id=$1 AND revision=$2 AND enabled`, id, *pending.ActivityRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	if err := resetActivityCoverage(ctx, tx, pending.ConnectionID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE bank_link_attempts SET state='completed',token_cipher=NULL WHERE id=$1 AND state='exchanging' AND purpose='activity'`, pending.ID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ActivityLink uses official update mode to request Transactions consent on an
// existing Item. Ordinary balance Link never requests this additional product.
func (p *sdkProvider) ActivityLink(ctx context.Context, user, token string) (string, time.Time, error) {
	request := plaid.NewLinkTokenCreateRequest("Bank Piggy", "en", []plaid.CountryCode{plaid.COUNTRYCODE_US})
	request.User = plaid.NewLinkTokenCreateRequestUser(user)
	request.SetRedirectUri(p.config.RedirectURI)
	request.SetWebhook(p.config.WebhookURL)
	request.SetAccessToken(token)
	request.Update = &plaid.LinkTokenCreateRequestUpdate{AccountSelectionEnabled: ptr(true)}
	request.AdditionalConsentedProducts = []plaid.Products{plaid.PRODUCTS_TRANSACTIONS}
	result, response, err := p.api.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(*request).Execute()
	if closeErr := closeResponse(response); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", time.Time{}, providerError(err)
	}
	return result.LinkToken, result.Expiration, nil
}
