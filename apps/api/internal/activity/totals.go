package activity

import "bankpiggy/api/internal/budget"

func totalRecords(totals *budget.ActivityTotals, records []budget.ActivityRecord) error {
	for _, record := range records {
		if countedRecord(record) {
			if err := addRecordTotals(totals, record, *record.ScopeAmountCents); err != nil {
				return err
			}
		}
	}
	var err error
	totals.NetInvestmentContributionsCents, err = addCents(totals.InvestmentDepositsCents, -totals.InvestmentWithdrawalsCents)
	return err
}

func countedRecord(record budget.ActivityRecord) bool {
	return record.ScopeAmountCents != nil && record.Category != "review" && record.Category != "excluded" && !record.Pending && !record.Cancelled
}

func addRecordTotals(totals *budget.ActivityTotals, record budget.ActivityRecord, cents int64) error {
	var target *int64
	if record.Kind == "bank" {
		switch record.Category {
		case "paycheck", "income":
			target, cents = &totals.IncomeCents, -cents
		case "spending":
			target = &totals.SpendingCents
		}
	} else {
		switch record.Category {
		case "contribution":
			target, cents = &totals.InvestmentDepositsCents, absolute(cents)
		case "withdrawal":
			target, cents = &totals.InvestmentWithdrawalsCents, absolute(cents)
		case "investment_income":
			target, cents = &totals.InvestmentIncomeCents, -cents
		case "spending":
			target = &totals.InvestmentFeesCents
		}
	}
	if target == nil {
		return nil
	}
	value, err := addCents(*target, cents)
	if err == nil {
		*target = value
	}
	return err
}

func addCents(first, second int64) (int64, error) {
	if second > maxSafeCents || second < -maxSafeCents || (second > 0 && first > maxSafeCents-second) || (second < 0 && first < -maxSafeCents-second) {
		return 0, ErrInvalid
	}
	return first + second, nil
}
