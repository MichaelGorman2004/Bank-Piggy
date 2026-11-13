package banking

import (
	"errors"
	plaid "github.com/plaid/plaid-go/v50/legacy"
)

// ProviderError contains only a fixed safe code, never provider payloads or credentials.
type ProviderError struct{ Code string }

func (e *ProviderError) Error() string { return e.Code }

func providerError(err error) error {
	if err == nil {
		return nil
	}
	var sdk plaid.GenericOpenAPIError
	if errors.As(err, &sdk) {
		if model, ok := sdk.Model().(plaid.PlaidError); ok {
			switch model.ErrorCode {
			case "ITEM_LOGIN_REQUIRED", "INVALID_CREDENTIALS", "INSUFFICIENT_CREDENTIALS", "USER_PERMISSION_REVOKED":
				return &ProviderError{Code: "reconnect_required"}
			case "INVALID_ACCESS_TOKEN", "ITEM_NOT_FOUND":
				return &ProviderError{Code: "item_removed"}
			}
		}
	}
	return &ProviderError{Code: "provider_unavailable"}
}
