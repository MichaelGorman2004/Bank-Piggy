package banking

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"bankpiggy/api/internal/budget"
)

func TestSnapshotPreservesOtherAmountsWhenOneHasFractionalCents(t *testing.T) {
	cases := []struct {
		name, current, available   string
		wantCurrent, wantAvailable *int64
	}{
		{"current unavailable", "0.001", "0.25", nil, ptr(int64(25))},
		{"available unavailable", "0.25", "0.001", ptr(int64(25)), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"accounts":[{"account_id":"exact","type":"investment","balances":{"current":0.29,"iso_currency_code":"USD"}},{"account_id":"fractional","type":"investment","balances":{"current":` + tc.current + `,"available":` + tc.available + `,"iso_currency_code":"USD"}}],"item":{}}`
			snapshot, _, err := decodeSnapshot(&http.Response{Body: io.NopCloser(strings.NewReader(body))})
			if err != nil || len(snapshot.Accounts) != 2 {
				t.Fatal("one fractional field rejected or removed account rows", err)
			}
			first, second := snapshot.Accounts[0].Account, snapshot.Accounts[1].Account
			if first.CurrentBalanceCents == nil || *first.CurrentBalanceCents != 29 {
				t.Fatal("exact amount in another account changed")
			}
			assertBalanceValue(t, second.CurrentBalanceCents, tc.wantCurrent)
			assertBalanceValue(t, second.AvailableBalanceCents, tc.wantAvailable)
			assertUnavailableSnapshotExcluded(t, first, second)
		})
	}
}

func assertBalanceValue(t *testing.T, got, want *int64) {
	t.Helper()
	if (got == nil) != (want == nil) || got != nil && *got != *want {
		t.Fatal("provider amount was rounded, replaced by zero, or lost")
	}
}

func assertUnavailableSnapshotExcluded(t *testing.T, first, second budget.ConnectedAccount) {
	t.Helper()
	plan := budget.Blank("First")
	plan.Members[0].Accounts = []budget.AccountBalance{{ID: "manual", Name: "Manual", Type: "savings", BalanceCents: 500}}
	first.Mapping = budget.AccountMapping{Enabled: true, Ownership: "member-1", Classification: "brokerage"}
	second.Mapping = budget.AccountMapping{Enabled: true, Ownership: "member-1", Classification: "brokerage", ManualMemberID: ptr("member-1"), ManualAccountID: ptr("manual")}
	result := EffectiveBalances(plan, []budget.Connection{{State: "connected", Accounts: []budget.ConnectedAccount{first, second}}})
	if second.CurrentBalanceCents == nil && (result.Accounts[1].Included || result.Household.InvestmentsCents != 29 || result.Household.SavingsCents != 500) {
		t.Fatal("unavailable current balance entered totals or replaced the saved manual balance")
	}
	if second.CurrentBalanceCents != nil && (!result.Accounts[1].Included || result.Household.InvestmentsCents != 54 || result.Household.SavingsCents != 0) {
		t.Fatal("unavailable available balance removed an exact current balance from totals")
	}
}

func TestFractionalSentinelDoesNotHideMalformedOrOutOfRangeAmounts(t *testing.T) {
	fraction := json.Number("0.001")
	if _, err := MoneyCents(&fraction); !errors.Is(err, errFractionalCents) {
		t.Fatal("exact money conversion no longer rejects fractional cents")
	}
	for _, text := range []string{"NaN", "1000000000.01", "1000000000.001", "-1000000000.001", "9223372036854775808", "1e1000000"} {
		number := json.Number(text)
		if _, err := snapshotBalance(&number); err == nil || errors.Is(err, errFractionalCents) {
			t.Fatal("invalid provider amount was hidden as unsupported precision")
		}
	}
}

func TestSnapshotStillRejectsInvalidOrOutOfRangeAmounts(t *testing.T) {
	for _, amount := range []string{"0.0.1", "1000000000.001", "-1000000000.001", "1e1000000"} {
		body := `{"accounts":[{"account_id":"simulated","type":"investment","balances":{"current":` + amount + `,"iso_currency_code":"USD"}}],"item":{}}`
		if _, _, err := decodeSnapshot(&http.Response{Body: io.NopCloser(strings.NewReader(body))}); err == nil {
			t.Fatal("invalid or out-of-range amount did not reject snapshot")
		}
	}
}
