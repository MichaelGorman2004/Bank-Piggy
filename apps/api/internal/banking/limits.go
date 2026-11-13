package banking

import "time"

const (
	providerConcurrency    = 2
	workerPollInterval     = 15 * time.Second
	workerJobTimeout       = 45 * time.Second
	minimumRefreshInterval = SyncInterval
	// Plaid's cached balance age is independent of how often Bank Piggy reads it.
	maximumCachedBalanceAge = 24 * time.Hour
	unknownKeyFetchInterval = 5 * time.Second
	webhookClockTolerance   = 30 * time.Second
	expiryCleanupBatch      = 100
	maxProviderAccounts     = 100
	providerNumberLimit     = 64
	webhookReceiptRetention = 24 * time.Hour
)
