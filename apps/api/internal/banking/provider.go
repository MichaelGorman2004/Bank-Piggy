package banking

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	plaid "github.com/plaid/plaid-go/v50/legacy"
)

const providerResponseLimit = 2 * 1024 * 1024

// Provider is the read-only provider boundary; simulations implement it only in tests.
type Provider interface {
	Link(context.Context, string, string, string) (string, time.Time, error)
	Exchange(context.Context, string) (string, string, error)
	Accounts(context.Context, string, string) (Snapshot, error)
	Remove(context.Context, string) error
	VerificationKey(context.Context, string) ([]byte, error)
}

type sdkProvider struct {
	api    *plaid.APIClient
	config Config
}
type boundedTransport struct{ next http.RoundTripper }

func (t boundedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := t.next.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	response.Body = &boundedBody{ReadCloser: response.Body, remaining: providerResponseLimit}
	return response, nil
}

type boundedBody struct {
	io.ReadCloser
	remaining int64
}

func (b *boundedBody) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		return 0, errors.New("provider response exceeds limit")
	}
	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.ReadCloser.Read(p)
	b.remaining -= int64(n)
	return n, err
}

// NewProvider uses the maintained official SDK with fixed endpoints and bounded transport.
func NewProvider(c Config) Provider {
	settings := plaid.NewConfiguration()
	settings.AddDefaultHeader("PLAID-CLIENT-ID", c.ClientID)
	settings.AddDefaultHeader("PLAID-SECRET", c.Secret)
	environment := plaid.Sandbox
	if c.Environment == "production" {
		environment = plaid.Production
	}
	settings.UseEnvironment(environment)
	settings.HTTPClient = &http.Client{Timeout: ProviderTimeout, Transport: boundedTransport{next: http.DefaultTransport}, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("provider redirects are disabled") }}
	return &sdkProvider{api: plaid.NewAPIClient(settings), config: c}
}

func closeResponse(response *http.Response) error {
	if response == nil || response.Body == nil {
		return nil
	}
	return response.Body.Close()
}

func (p *sdkProvider) Link(ctx context.Context, user, kind, token string) (string, time.Time, error) {
	request := plaid.NewLinkTokenCreateRequest("Bank Piggy", "en", []plaid.CountryCode{plaid.COUNTRYCODE_US})
	request.User = plaid.NewLinkTokenCreateRequestUser(user)
	request.SetRedirectUri(p.config.RedirectURI)
	request.SetWebhook(p.config.WebhookURL)
	if token != "" {
		request.SetAccessToken(token)
		request.Update = &plaid.LinkTokenCreateRequestUpdate{AccountSelectionEnabled: ptr(true)}
	} else {
		configureProducts(request, kind)
	}
	result, response, err := p.api.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(*request).Execute()
	if closeErr := closeResponse(response); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", time.Time{}, providerError(err)
	}
	return result.LinkToken, result.Expiration, nil
}

func configureProducts(request *plaid.LinkTokenCreateRequest, kind string) {
	if kind == "investment" {
		request.Products = []plaid.Products{plaid.PRODUCTS_INVESTMENTS}
		return
	}
	request.Products = []plaid.Products{plaid.PRODUCTS_AUTH}
	disabled := false
	request.Auth = &plaid.LinkTokenCreateRequestAuth{AuthTypeSelectEnabled: &disabled, InstantMatchEnabled: &disabled, AutomatedMicrodepositsEnabled: &disabled, SameDayMicrodepositsEnabled: &disabled, InstantMicrodepositsEnabled: &disabled, DatabaseMatchEnabled: &disabled, DatabaseInsightsEnabled: &disabled, SmsMicrodepositsVerificationEnabled: &disabled}
}

func (p *sdkProvider) Exchange(ctx context.Context, public string) (string, string, error) {
	result, response, err := p.api.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(*plaid.NewItemPublicTokenExchangeRequest(public)).Execute()
	if closeErr := closeResponse(response); err == nil {
		err = closeErr
	}
	return result.AccessToken, result.ItemId, providerError(err)
}

func (p *sdkProvider) Remove(ctx context.Context, token string) error {
	_, response, err := p.api.PlaidApi.ItemRemove(ctx).ItemRemoveRequest(*plaid.NewItemRemoveRequest(token)).Execute()
	if closeErr := closeResponse(response); err == nil {
		err = closeErr
	}
	return providerError(err)
}

func (p *sdkProvider) VerificationKey(ctx context.Context, id string) ([]byte, error) {
	result, response, err := p.api.PlaidApi.WebhookVerificationKeyGet(ctx).WebhookVerificationKeyGetRequest(*plaid.NewWebhookVerificationKeyGetRequest(id)).Execute()
	if closeErr := closeResponse(response); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, providerError(err)
	}
	return json.Marshal(result.Key)
}

func ptr[T any](value T) *T { return &value }
