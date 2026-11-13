package banking

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"bankpiggy/api/internal/budget"
	plaid "github.com/plaid/plaid-go/v50/legacy"
)

// Snapshot is a full authoritative account list; missing accounts become inactive.
type Snapshot struct {
	Accounts        []ProviderAccount
	InstitutionName string
}

// ProviderAccount retains the stable provider identifier only on the server.
type ProviderAccount struct {
	ProviderID string
	Account    budget.ConnectedAccount
}

type rawAccounts struct {
	Accounts []struct {
		ID       string  `json:"account_id"`
		Name     string  `json:"name"`
		Mask     *string `json:"mask"`
		Type     string  `json:"type"`
		Subtype  *string `json:"subtype"`
		Balances struct {
			Current   *json.Number `json:"current"`
			Available *json.Number `json:"available"`
			Currency  *string      `json:"iso_currency_code"`
			Updated   *string      `json:"last_updated_datetime"`
		} `json:"balances"`
	} `json:"accounts"`
	Item struct {
		InstitutionID *string `json:"institution_id"`
	} `json:"item"`
}

var providerNumber = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]{1,2})?$`)
var errFractionalCents = errors.New("provider money is not whole cents")

// MoneyCents converts provider decimal JSON exactly; fractional cents are rejected.
func MoneyCents(number *json.Number) (*int64, error) {
	if number == nil {
		return nil, nil
	}
	text := number.String()
	if len(text) > providerNumberLimit || !providerNumber.MatchString(text) {
		return nil, errors.New("provider money exceeds range")
	}
	rational, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, errors.New("invalid provider money")
	}
	rational.Mul(rational, big.NewRat(100, 1))
	limit := big.NewRat(budget.MaxMoneyCents, 1)
	if rational.Cmp(limit) > 0 || rational.Cmp(new(big.Rat).Neg(limit)) < 0 {
		return nil, errors.New("provider money exceeds range")
	}
	if !rational.IsInt() {
		return nil, errFractionalCents
	}
	cents := rational.Num().Int64()
	return &cents, nil
}

// snapshotBalance leaves fractional-cent amounts unavailable without rounding.
// Malformed and out-of-range amounts still reject the authoritative snapshot.
func snapshotBalance(number *json.Number) (*int64, error) {
	cents, err := MoneyCents(number)
	if errors.Is(err, errFractionalCents) {
		return nil, nil
	}
	return cents, err
}

func decodeSnapshot(response *http.Response) (Snapshot, string, error) {
	var raw rawAccounts
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return Snapshot{}, "", err
	}
	if len(raw.Accounts) > maxProviderAccounts {
		return Snapshot{}, "", errors.New("too many provider accounts")
	}
	result := Snapshot{Accounts: make([]ProviderAccount, 0, len(raw.Accounts)), InstitutionName: "Connected institution"}
	seen := map[string]bool{}
	for _, value := range raw.Accounts {
		if value.ID == "" || seen[value.ID] {
			return Snapshot{}, "", errors.New("invalid provider account identifiers")
		}
		seen[value.ID] = true
		current, err := snapshotBalance(value.Balances.Current)
		if err != nil {
			return Snapshot{}, "", err
		}
		available, err := snapshotBalance(value.Balances.Available)
		if err != nil {
			return Snapshot{}, "", err
		}
		account := budget.ConnectedAccount{Name: strings.TrimSpace(value.Name), ProviderType: value.Type, ProviderSubtype: value.Subtype, Mask: value.Mask, Currency: value.Balances.Currency, CurrentBalanceCents: current, AvailableBalanceCents: available, ProviderUpdatedAt: value.Balances.Updated, Active: true}
		result.Accounts = append(result.Accounts, ProviderAccount{ProviderID: value.ID, Account: account})
	}
	institution := ""
	if raw.Item.InstitutionID != nil {
		institution = *raw.Item.InstitutionID
	}
	return result, institution, nil
}

func (p *sdkProvider) Accounts(ctx context.Context, token, kind string) (Snapshot, error) {
	response, err := p.accountResponse(ctx, token, kind)
	if err != nil {
		if closeErr := closeResponse(response); closeErr != nil {
			return Snapshot{}, providerError(err)
		}
		return Snapshot{}, providerError(err)
	}
	snapshot, institution, err := decodeSnapshot(response)
	if closeErr := closeResponse(response); err == nil {
		err = closeErr
	}
	if err != nil {
		return Snapshot{}, err
	}
	if institution != "" {
		snapshot.InstitutionName = p.institutionName(ctx, institution)
	}
	return snapshot, nil
}

func (p *sdkProvider) institutionName(ctx context.Context, id string) string {
	result, response, err := p.api.PlaidApi.InstitutionsGetById(ctx).InstitutionsGetByIdRequest(*plaid.NewInstitutionsGetByIdRequest(id, []plaid.CountryCode{plaid.COUNTRYCODE_US})).Execute()
	if closeErr := closeResponse(response); err == nil {
		err = closeErr
	}
	if err != nil {
		return "Connected institution"
	}
	return result.Institution.Name
}

func (p *sdkProvider) accountResponse(ctx context.Context, token, kind string) (*http.Response, error) {
	if kind == "depository" {
		// Some institutions require a freshness bound for their cached-only accounts.
		oldest := time.Now().UTC().Add(-maximumCachedBalanceAge).Truncate(time.Second)
		request := plaid.NewAccountsBalanceGetRequest(token)
		request.Options = &plaid.AccountsBalanceGetRequestOptions{MinLastUpdatedDatetime: &oldest}
		_, response, err := p.api.PlaidApi.AccountsBalanceGet(ctx).AccountsBalanceGetRequest(*request).Execute()
		return response, err
	}
	_, response, err := p.api.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*plaid.NewAccountsGetRequest(token)).Execute()
	return response, err
}
