package banking

import "time"

const (
	providerConcurrency     = 2
	workerPollInterval      = 15 * time.Second
	workerJobTimeout        = 45 * time.Second
	minimumRefreshInterval  = time.Hour
	maximumCachedBalanceAge = SyncInterval
	unknownKeyFetchInterval = 5 * time.Second
	webhookClockTolerance   = 30 * time.Second
	expiryCleanupBatch      = 100
	maxProviderAccounts     = 100
	providerNumberLimit     = 64
	webhookReceiptRetention = 24 * time.Hour
)
