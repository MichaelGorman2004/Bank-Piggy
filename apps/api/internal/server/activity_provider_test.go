package server

import (
	"context"
	"errors"
	"time"

	"bankpiggy/api/internal/banking"
)

func (*httpBankProvider) ActivityLink(context.Context, string, string) (string, time.Time, error) {
	return "", time.Time{}, errors.New("HTTP boundary simulation does not initialize activity products")
}

func (*httpBankProvider) TransactionsSync(context.Context, string, string) (banking.TransactionPage, error) {
	return banking.TransactionPage{}, errors.New("HTTP boundary simulation does not retrieve activity")
}

func (*httpBankProvider) InvestmentTransactions(context.Context, string, string, string, int) (banking.InvestmentPage, error) {
	return banking.InvestmentPage{}, errors.New("HTTP boundary simulation does not retrieve investment activity")
}
