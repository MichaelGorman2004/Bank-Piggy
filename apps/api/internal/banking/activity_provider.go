package banking

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"bankpiggy/api/internal/budget"
	plaid "github.com/plaid/plaid-go/v50/legacy"
)

const activityPageSize = 500
const maxActivityRecords = 20000
const maxActivityPages = 100

// ProviderActivityRecord keeps provider identifiers and cancellation links internal.
type ProviderActivityRecord struct {
	ProviderID, ProviderAccountID, CancelledProviderID string
	Record                                             budget.ActivityRecord
}

// TransactionPage is one page of an atomic cursor update, not a saved checkpoint.
type TransactionPage struct {
	Records                  []ProviderActivityRecord
	Removed                  []string
	NextCursor               string
	HasMore, HistoryComplete bool
}

// InvestmentPage is one bounded offset page. Offsets provide no snapshot token;
// stable totals and unique identifiers detect some, but not all, concurrent changes.
type InvestmentPage struct {
	Records []ProviderActivityRecord
	Total   int
}

type rawActivity struct {
	ID              string       `json:"transaction_id"`
	InvestmentID    string       `json:"investment_transaction_id"`
	AccountID       string       `json:"account_id"`
	CancelID        *string      `json:"cancel_transaction_id"`
	Date            string       `json:"date"`
	Name            string       `json:"name"`
	Merchant        *string      `json:"merchant_name"`
	Amount          *json.Number `json:"amount"`
	Currency        *string      `json:"iso_currency_code"`
	Pending         bool         `json:"pending"`
	Type            string       `json:"type"`
	Subtype         string       `json:"subtype"`
	TransactionType string       `json:"transaction_type"`
	Category        struct {
		Primary    string `json:"primary"`
		Detailed   string `json:"detailed"`
		Confidence string `json:"confidence_level"`
	} `json:"personal_finance_category"`
}

type rawTransactionPage struct {
	Added    []rawActivity `json:"added"`
	Modified []rawActivity `json:"modified"`
	Removed  []struct {
		ID string `json:"transaction_id"`
	} `json:"removed"`
	NextCursor *string `json:"next_cursor"`
	HasMore    *bool   `json:"has_more"`
	Status     string  `json:"transactions_update_status"`
}

type rawInvestmentPage struct {
	Records []rawActivity `json:"investment_transactions"`
	Total   *int          `json:"total_investment_transactions"`
}

func normalizeActivity(raw rawActivity, kind string) (ProviderActivityRecord, error) {
	id := raw.ID
	if kind == "investment" {
		id = raw.InvestmentID
	}
	if id == "" || raw.AccountID == "" {
		return ProviderActivityRecord{}, errors.New("invalid activity identifiers")
	}
	if _, err := time.Parse(time.DateOnly, raw.Date); err != nil {
		return ProviderActivityRecord{}, errors.New("invalid activity date")
	}
	cents, err := snapshotBalance(raw.Amount)
	if err != nil {
		return ProviderActivityRecord{}, err
	}
	record := budget.ActivityRecord{Kind: kind, Date: budget.CalendarDate(raw.Date), Name: raw.Name, Currency: raw.Currency, AmountCents: cents, ActivityType: raw.Type, ActivitySubtype: raw.Subtype, PrimaryCategory: raw.Category.Primary, DetailedCategory: raw.Category.Detailed, CategoryConfidence: raw.Category.Confidence, Pending: raw.Pending}
	if kind == "bank" {
		record.ActivityType = raw.TransactionType
	}
	if raw.Merchant != nil {
		record.Counterparty = *raw.Merchant
	}
	value := ProviderActivityRecord{ProviderID: id, ProviderAccountID: raw.AccountID, Record: record}
	if raw.CancelID != nil {
		value.CancelledProviderID = *raw.CancelID
	}
	if raw.Type == "cancel" {
		value.Record.Cancelled = true
	}
	return value, nil
}

func decodeActivityResponse(response *http.Response, original error, target any) error {
	if original != nil {
		return providerError(original)
	}
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	return decoder.Decode(target)
}

// TransactionsSync returns exact decimal values through bounded raw decoding of
// the official SDK response. It never rounds fractional cents to zero.
func (p *sdkProvider) TransactionsSync(ctx context.Context, token, cursor string) (TransactionPage, error) {
	response, err := p.transactionResponse(ctx, token, cursor)
	var raw rawTransactionPage
	err = decodeActivityResponse(response, err, &raw)
	if closeErr := closeResponse(response); err == nil {
		err = closeErr
	}
	if err != nil {
		return TransactionPage{}, err
	}
	if err := validateTransactionPage(raw, cursor); err != nil {
		return TransactionPage{}, err
	}
	result := TransactionPage{NextCursor: *raw.NextCursor, HasMore: *raw.HasMore, HistoryComplete: raw.Status == "HISTORICAL_UPDATE_COMPLETE"}
	for _, value := range append(raw.Added, raw.Modified...) {
		record, err := normalizeActivity(value, "bank")
		if err != nil {
			return result, err
		}
		result.Records = append(result.Records, record)
	}
	for _, value := range raw.Removed {
		if value.ID == "" {
			return result, errors.New("invalid removed activity identifier")
		}
		result.Removed = append(result.Removed, value.ID)
	}
	return result, nil
}

func validateTransactionPage(raw rawTransactionPage, savedCursor string) error {
	if raw.Added == nil || raw.Modified == nil || raw.Removed == nil || raw.NextCursor == nil || raw.HasMore == nil {
		return errors.New("invalid transaction history response")
	}
	if !usableTransactionCursor(raw, savedCursor) {
		return errors.New("transaction checkpoint was omitted")
	}
	return nil
}

func usableTransactionCursor(raw rawTransactionPage, saved string) bool {
	if *raw.NextCursor != "" {
		return true
	}
	// An empty first cursor means not-ready history, never a completed empty ledger.
	return saved == "" && raw.Status != "HISTORICAL_UPDATE_COMPLETE" && len(raw.Added)+len(raw.Modified)+len(raw.Removed) == 0
}

// InvestmentTransactions returns one exact, bounded page for a fixed date window.
func (p *sdkProvider) InvestmentTransactions(ctx context.Context, token, start, end string, offset int) (InvestmentPage, error) {
	response, err := p.investmentResponse(ctx, token, start, end, offset)
	var raw rawInvestmentPage
	err = decodeActivityResponse(response, err, &raw)
	if closeErr := closeResponse(response); err == nil {
		err = closeErr
	}
	if err != nil {
		return InvestmentPage{}, err
	}
	if raw.Records == nil || raw.Total == nil {
		return InvestmentPage{}, errors.New("invalid investment history response")
	}
	result := InvestmentPage{Total: *raw.Total}
	for _, value := range raw.Records {
		record, err := normalizeActivity(value, "investment")
		if err != nil {
			return result, err
		}
		result.Records = append(result.Records, record)
	}
	return result, nil
}

func (p *sdkProvider) transactionResponse(ctx context.Context, token, cursor string) (*http.Response, error) {
	request := plaid.NewTransactionsSyncRequest(token)
	request.Cursor = ptr(cursor)
	request.Count = ptr(int32(activityPageSize))
	if cursor == "" {
		request.Options = &plaid.TransactionsSyncRequestOptions{DaysRequested: ptr(int32(activityHistoryDays))}
	}
	_, response, err := p.api.PlaidApi.TransactionsSync(ctx).TransactionsSyncRequest(*request).Execute()
	return response, err
}

func (p *sdkProvider) investmentResponse(ctx context.Context, token, start, end string, offset int) (*http.Response, error) {
	if offset < 0 || offset > maxActivityRecords {
		return nil, ErrInvalid
	}
	request := plaid.NewInvestmentsTransactionsGetRequest(token, start, end)
	request.Options = &plaid.InvestmentsTransactionsGetRequestOptions{Count: ptr(int32(activityPageSize)), Offset: ptr(int32(offset))}
	_, response, err := p.api.PlaidApi.InvestmentsTransactionsGet(ctx).InvestmentsTransactionsGetRequest(*request).Execute()
	return response, err
}
