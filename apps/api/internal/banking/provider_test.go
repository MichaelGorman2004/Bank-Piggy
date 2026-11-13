package banking

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	plaid "github.com/plaid/plaid-go/v50/legacy"
)

type providerTransportFunc func(*http.Request) (*http.Response, error)

func (f providerTransportFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestProviderRoutesCredentialsToOfficialEnvironment(t *testing.T) {
	for environment, host := range map[string]string{"sandbox": "sandbox.plaid.com", "production": "production.plaid.com"} {
		t.Run(environment, func(t *testing.T) {
			config := testConfig()
			config.Environment = environment
			provider, ok := NewProvider(config).(*sdkProvider)
			if !ok {
				t.Fatal("official provider was not constructed")
			}
			provider.api.GetConfig().HTTPClient.Transport = providerDestinationTransport(t, config, host)
			if token, _, err := provider.Link(t.Context(), "simulated-user", "depository", ""); err != nil || token != "simulated-link" {
				t.Fatal("provider did not complete the Link request", err)
			}
		})
	}
}

func providerDestinationTransport(t *testing.T, config Config, host string) http.RoundTripper {
	t.Helper()
	return providerTransportFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Scheme != "https" || request.URL.Host != host || request.URL.Path != "/link/token/create" {
			t.Fatal("provider credentials routed to the wrong endpoint")
		}
		if request.Header.Get("PLAID-CLIENT-ID") != config.ClientID || request.Header.Get("PLAID-SECRET") != config.Secret {
			t.Fatal("server credentials absent from provider request")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"link_token":"simulated-link","expiration":"2030-01-01T00:00:00Z","request_id":"simulated"}`))}, nil
	})
}

func TestProviderDoesNotForwardCredentialsOnRedirect(t *testing.T) {
	provider, ok := NewProvider(testConfig()).(*sdkProvider)
	if !ok {
		t.Fatal("official provider was not constructed")
	}
	calls := 0
	provider.api.GetConfig().HTTPClient.Transport = providerTransportFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.URL.Host != "sandbox.plaid.com" {
			t.Fatal("provider followed a redirect to another destination")
		}
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{"Location": {"https://foreign.example/link/token/create"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	_, _, err := provider.Link(t.Context(), "simulated-user", "depository", "")
	if err == nil || calls != 1 {
		t.Fatal("provider redirect was not rejected")
	}
}

func sdkSimulation(t *testing.T, handler http.HandlerFunc) *sdkProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	config := plaid.NewConfiguration()
	config.Servers = plaid.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = &http.Client{Timeout: ProviderTimeout, Transport: boundedTransport{next: http.DefaultTransport}}
	return &sdkProvider{api: plaid.NewAPIClient(config), config: testConfig()}
}

func TestOfficialSDKReadsRawExactMoneyAndMissingBalances(t *testing.T) {
	fixture := `{"accounts":[{"account_id":"simulated-account","name":"Simulated account","type":"depository","subtype":"savings","mask":"1234","balances":{"current":0.29,"available":null,"iso_currency_code":"USD"}}],"item":{"item_id":"simulated-item","institution_id":null},"request_id":"simulated-request"}`
	provider := sdkSimulation(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/balance/get" {
			t.Errorf("unexpected provider operation %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(fixture)); err != nil {
			t.Error(err)
		}
	})
	snapshot, err := provider.Accounts(t.Context(), "access-simulated", "depository")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Accounts) != 1 || *snapshot.Accounts[0].Account.CurrentBalanceCents != 29 || snapshot.Accounts[0].Account.AvailableBalanceCents != nil {
		t.Fatal("SDK float conversion or missing balance changed the exact provider amount")
	}
}

func TestOfficialSDKProviderErrorsAreSafeAndActionable(t *testing.T) {
	provider := sdkSimulation(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		if _, err := w.Write([]byte(`{"error_type":"ITEM_ERROR","error_code":"ITEM_LOGIN_REQUIRED","error_message":"access-simulated-secret","request_id":"simulated-request"}`)); err != nil {
			t.Error(err)
		}
	})
	_, err := provider.Accounts(t.Context(), "access-simulated", "investment")
	var safe *ProviderError
	if !errors.As(err, &safe) || safe.Code != "reconnect_required" || err.Error() != "reconnect_required" {
		t.Fatal("official SDK error did not produce a safe reconnect state", err)
	}
}

func TestDepositoryLinkDisablesManualAuthFlows(t *testing.T) {
	request := plaid.NewLinkTokenCreateRequest("Bank Piggy", "en", []plaid.CountryCode{plaid.COUNTRYCODE_US})
	configureProducts(request, "depository")
	if len(request.Products) != 1 || request.Products[0] != plaid.PRODUCTS_AUTH || request.Auth == nil {
		t.Fatal("depository Link does not request only Auth")
	}
	data, err := json.Marshal(request.Auth)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]bool
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	for key, enabled := range fields {
		if enabled {
			t.Fatalf("manual Auth flow enabled: %s", key)
		}
	}
}

func TestInvestmentLinkUsesOneReadOnlyProduct(t *testing.T) {
	request := plaid.NewLinkTokenCreateRequest("Bank Piggy", "en", []plaid.CountryCode{plaid.COUNTRYCODE_US})
	configureProducts(request, "investment")
	if len(request.Products) != 1 || request.Products[0] != plaid.PRODUCTS_INVESTMENTS {
		t.Fatal("investment Link requests unrelated product")
	}
}

func TestProviderLeavesUnsupportedFractionalCentsUnavailable(t *testing.T) {
	provider := sdkSimulation(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"accounts":[{"account_id":"simulated-account","name":"Simulated","type":"investment","balances":{"current":0.001,"available":null,"iso_currency_code":"USD"}}],"item":{},"request_id":"simulated"}`)); err != nil {
			t.Error(err)
		}
	})
	snapshot, err := provider.Accounts(context.Background(), "simulated-token", "investment")
	if err != nil || len(snapshot.Accounts) != 1 || snapshot.Accounts[0].Account.CurrentBalanceCents != nil {
		t.Fatal("fractional cents were rounded, treated as zero, or removed the account", err)
	}
}
