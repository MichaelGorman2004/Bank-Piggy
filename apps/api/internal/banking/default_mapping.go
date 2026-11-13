package banking

import "bankpiggy/api/internal/budget"

const initialMappingRevision int64 = 1

// defaultAccountMapping resolves untouched imports without saving a user decision.
// Only provider type fields determine classification; manual matching stays explicit.
// Missing current balances keep their type but remain outside effective totals.
func defaultAccountMapping(account budget.ConnectedAccount, owner string) budget.AccountMapping {
	if !untouchedAccountMapping(account) || !account.Active || owner == "" || account.Currency == nil || *account.Currency != "USD" {
		return account.Mapping
	}
	kind := defaultAccountClassification(account)
	if kind == "unclassified" {
		return account.Mapping
	}
	return budget.AccountMapping{Enabled: true, Ownership: budget.AccountOwnership(owner), Classification: kind}
}

func untouchedAccountMapping(account budget.ConnectedAccount) bool {
	mapping := account.Mapping
	return account.MappingRevision == initialMappingRevision && !mapping.Enabled && mapping.Classification == "unclassified" && mapping.ManualMemberID == nil && mapping.ManualAccountID == nil
}

func defaultAccountClassification(account budget.ConnectedAccount) budget.ConnectedAccountType {
	if !supportedAccount(account) {
		return "unclassified"
	}
	subtype := ""
	if account.ProviderSubtype != nil {
		subtype = *account.ProviderSubtype
	}
	if subtype == "hsa" {
		return "hsa"
	}
	if account.ProviderType == "investment" {
		if subtype == "brokerage" {
			return "brokerage"
		}
		return "other_investment"
	}
	switch subtype {
	case "checking":
		return "checking"
	case "savings":
		return "savings"
	default:
		return "unclassified"
	}
}
