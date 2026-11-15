package banking

import (
	"testing"
	"time"

	"bankpiggy/api/internal/budget"
)

func TestActivitySourceAvailabilityUsesItsHistoryProduct(t *testing.T) {
	cases := []struct {
		kind, account string
		want          bool
	}{
		{"depository", "depository", true}, {"depository", "credit", true}, {"depository", "investment", false},
		{"investment", "investment", true}, {"investment", "depository", false}, {"investment", "credit", false},
	}
	for _, tc := range cases {
		account := budget.ConnectedAccount{Active: true, Currency: ptr("USD"), ProviderType: tc.account}
		if got := activityAccountAvailable(account, tc.kind); got != tc.want {
			t.Errorf("%s/%s: %v", tc.kind, tc.account, got)
		}
	}
}

func TestRevokedSelectedAccountRemainsAnUnavailableCoverageObligation(t *testing.T) {
	s, identity, connection := serviceActivityFixture(t, false)
	execSQL(t, s, `UPDATE bank_activity_jobs SET history_complete=true,history_start='2026-10-01',history_end=CURRENT_DATE WHERE connection_id=$1`, connection.ID)
	execSQL(t, s, `DELETE FROM bank_accounts WHERE id=$1`, connection.Accounts[0].ID)
	month := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	result, err := s.MonthlyActivity(t.Context(), identity, month, "household")
	if err != nil {
		t.Fatal(err)
	}
	if result.Coverage != "incomplete" || len(result.Sources[0].Accounts) != 1 {
		t.Fatal("revoked account vanished from coverage")
	}
	account := result.Sources[0].Accounts[0]
	if !account.Enabled || account.Available || account.Name != "Unavailable account" {
		t.Fatal("revoked selection did not retain a safe unavailable placeholder")
	}
	request := budget.ActivitySettingsRequest{ExpectedRevision: result.Sources[0].Revision, Enabled: true, Accounts: []budget.ActivityAccountSelection{{AccountID: account.AccountID, Enabled: false, Ownership: identity.MemberID}}}
	if _, err := s.SetActivitySettings(t.Context(), identity, connection.ID, request); err != nil {
		t.Fatal(err)
	}
	result, err = s.MonthlyActivity(t.Context(), identity, month, "household")
	if err != nil {
		t.Fatal(err)
	}
	if result.Coverage != "disabled" {
		t.Fatal("explicit opt-out did not clear the missing-account obligation")
	}
}
