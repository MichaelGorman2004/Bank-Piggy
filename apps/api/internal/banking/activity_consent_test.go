package banking

import (
	"errors"
	"testing"

	"bankpiggy/api/internal/budget"
	"github.com/jackc/pgx/v5"
)

func TestActivityRequiresOperatorGateAndSeparateBankConsent(t *testing.T) {
	s, _, identity, connection := connectedService(t)
	request := budget.ActivitySettingsRequest{Enabled: true, Accounts: []budget.ActivityAccountSelection{{AccountID: connection.Accounts[0].ID, Enabled: true, Ownership: identity.MemberID}}}
	if _, err := s.SetActivitySettings(t.Context(), identity, connection.ID, request); !errors.Is(err, ErrUnavailable) {
		t.Fatal("operator gate did not reject activation", err)
	}
	s.config.BankActivityApproved = true
	source, err := s.SetActivitySettings(t.Context(), identity, connection.ID, request)
	if err != nil || !source.NeedsConsent || source.State != "consent_required" {
		t.Fatal("bank activity bypassed consent", source, err)
	}
	if err := s.processActivity(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("unconsented job read provider", err)
	}
	balances, err := s.CreateLink(t.Context(), identity, budget.LinkTokenRequest{ConnectionID: &connection.ID})
	if err != nil {
		t.Fatal(err)
	}
	completeTestLink(t, s, identity, balances)
	if !activitySourcesForTest(t, s, identity)[0].NeedsConsent {
		t.Fatal("balance repair authorized Transactions")
	}
	link, err := s.CreateLink(t.Context(), identity, budget.LinkTokenRequest{ConnectionID: &connection.ID, Purpose: ptr(budget.LinkPurpose("activity"))})
	if err != nil {
		t.Fatal(err)
	}
	assertRecoveredActivityPurpose(t, s, identity, link)
	completeTestLink(t, s, identity, link)
	if activitySourcesForTest(t, s, identity)[0].NeedsConsent {
		t.Fatal("completed activity attempt did not authorize history")
	}
}

func assertRecoveredActivityPurpose(t *testing.T, s *Service, identity Identity, link budget.LinkTokenResponse) {
	t.Helper()
	recovered, err := s.PendingLink(t.Context(), identity)
	if err != nil || recovered.Purpose == nil || *recovered.Purpose != "activity" || recovered.AttemptID != link.AttemptID {
		t.Fatal("OAuth recovery lost activity purpose", err)
	}
	other := identity
	other.SessionHash = "different-session"
	if err := s.CompleteLink(t.Context(), other, budget.ExchangeRequest{AttemptID: link.AttemptID}); !errors.Is(err, ErrNotFound) {
		t.Fatal("another session completed consent", err)
	}
}

func TestLateActivityConsentPreservesNewSettings(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	request := budget.ActivitySettingsRequest{Enabled: true, Accounts: []budget.ActivityAccountSelection{{AccountID: connection.Accounts[0].ID, Enabled: true, Ownership: identity.MemberID}}}
	source, err := s.SetActivitySettings(t.Context(), identity, connection.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	link, err := s.CreateLink(t.Context(), identity, budget.LinkTokenRequest{ConnectionID: &connection.ID, Purpose: ptr(budget.LinkPurpose("activity"))})
	if err != nil {
		t.Fatal(err)
	}
	request.ExpectedRevision, request.Enabled = source.Revision, false
	if _, err := s.SetActivitySettings(t.Context(), identity, connection.ID, request); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteLink(t.Context(), identity, budget.ExchangeRequest{AttemptID: link.AttemptID}); !errors.Is(err, ErrConflict) {
		t.Fatal("old consent overwrote new settings", err)
	}
	saved := activitySourcesForTest(t, s, identity)[0]
	if saved.Enabled || saved.Revision != source.Revision+1 {
		t.Fatal("new settings were lost")
	}
	if err := s.processActivity(t.Context(), connection.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("disabled source was read", err)
	}
}

func TestActivitySettingsRejectOtherMemberStaleRevisionAndForeignAccounts(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	source := enableActivityForTest(t, s, identity, connection)
	request := budget.ActivitySettingsRequest{ExpectedRevision: source.Revision, Enabled: false, Accounts: []budget.ActivityAccountSelection{}}
	other := identity
	other.MemberID = "member-2"
	if _, err := s.SetActivitySettings(t.Context(), other, connection.ID, request); !errors.Is(err, ErrNotFound) {
		t.Fatal("another member changed source consent", err)
	}
	request.ExpectedRevision = 0
	if _, err := s.SetActivitySettings(t.Context(), identity, connection.ID, request); !errors.Is(err, ErrConflict) {
		t.Fatal("stale settings were accepted", err)
	}
	request.ExpectedRevision = source.Revision
	request.Accounts = []budget.ActivityAccountSelection{{AccountID: "foreign-account", Ownership: "member-1"}}
	if _, err := s.SetActivitySettings(t.Context(), identity, connection.ID, request); !errors.Is(err, ErrInvalid) {
		t.Fatal("foreign account accepted", err)
	}
}

func TestActivitySourceDefaultsKeepCreditAndSavedExclusionsSeparate(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	execSQL(t, s, `UPDATE bank_accounts SET snapshot=jsonb_set(snapshot,'{providerType}','"credit"') WHERE connection_id=$1`, connection.ID)
	source := activitySourcesForTest(t, s, identity)[0]
	if source.Revision != 0 || !source.Accounts[0].Enabled || !source.Accounts[0].Available {
		t.Fatal("credit inherited unsupported asset mapping")
	}
	execSQL(t, s, `UPDATE bank_accounts SET mapping_revision=2 WHERE connection_id=$1`, connection.ID)
	if activitySourcesForTest(t, s, identity)[0].Accounts[0].Enabled {
		t.Fatal("explicit balance exclusion was lost")
	}
	request := budget.ActivitySettingsRequest{Enabled: true, Accounts: []budget.ActivityAccountSelection{}}
	if _, err := s.SetActivitySettings(t.Context(), identity, connection.ID, request); err != nil {
		t.Fatal(err)
	}
	execSQL(t, s, `INSERT INTO bank_accounts(id,household_id,connection_id,provider_id,snapshot) SELECT 'new-account',household_id,connection_id,'new-provider-account',snapshot FROM bank_accounts WHERE connection_id=$1 LIMIT 1`, connection.ID)
	for _, account := range activitySourcesForTest(t, s, identity)[0].Accounts {
		if account.Enabled {
			t.Fatal("unsaved account silently entered activity")
		}
	}
}

func TestUntouchedCreditActivityBelongsToLinkingMember(t *testing.T) {
	s, _, identity, connection := activityTestService(t)
	identity.MemberID = "member-2"
	execSQL(t, s, `UPDATE bank_connections SET member_id='member-2' WHERE id=$1`, connection.ID)
	execSQL(t, s, `UPDATE bank_accounts SET snapshot=jsonb_set(snapshot,'{providerType}','"credit"') WHERE connection_id=$1`, connection.ID)
	source := activitySourcesForTest(t, s, identity)[0]
	if source.Accounts[0].Ownership != "member-2" || !source.CanManage {
		t.Fatal("untouched credit ownership followed the asset placeholder instead of linking member")
	}
}
