package budget

import "errors"

// MaxAccounts bounds each member's manually entered account snapshots.
const MaxAccounts = 20

func savingsAccount(kind AccountType) bool {
	return kind == "savings" || kind == "high_yield_savings"
}

func validAccountType(kind AccountType) bool {
	switch kind {
	case "traditional_401k", "roth_401k", "traditional_ira", "roth_ira", "brokerage", "high_yield_savings", "savings", "other_investment":
		return true
	default:
		return false
	}
}

func accountTotals(accounts []AccountBalance) (savings, investments int64) {
	for _, account := range accounts {
		if savingsAccount(account.Type) {
			savings += account.BalanceCents
		} else {
			investments += account.BalanceCents
		}
	}
	return savings, investments
}

func validateAccounts(accounts []AccountBalance) error {
	if accounts == nil || len(accounts) > MaxAccounts {
		return errors.New("accounts must be an array of at most 20 accounts")
	}
	seen := map[string]bool{}
	for _, account := range accounts {
		if seen[account.ID] {
			return errors.New("duplicate account ID")
		}
		if err := validateAccount(account); err != nil {
			return err
		}
		seen[account.ID] = true
	}
	savings, investments := accountTotals(accounts)
	if !validMoney(savings) || !validMoney(investments) {
		return errors.New("savings and investment totals must each be at most 100000000000 cents")
	}
	return nil
}

func validateAccount(account AccountBalance) error {
	if !validID.MatchString(account.ID) || !validText(account.Name, 100) || !validAccountType(account.Type) || !validMoney(account.BalanceCents) {
		return errors.New("invalid account ID, name, type, or balance")
	}
	return nil
}
