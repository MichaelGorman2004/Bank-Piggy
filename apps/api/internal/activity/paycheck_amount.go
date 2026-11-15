package activity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"bankpiggy/api/internal/budget"
)

const paycheckMonthsPerYear int64 = 12
const paycheckVariableBasisPoints int64 = 500
const paycheckBasisPoints int64 = 10000
const paycheckVariableMinimumCents int64 = 100

func paycheckAmounts(records []budget.ActivityRecord) ([]int64, int64) {
	amounts := make([]int64, 0, len(records))
	for _, record := range records {
		amounts = append(amounts, -*record.AmountCents)
	}
	sort.Slice(amounts, func(i, j int) bool { return amounts[i] < amounts[j] })
	middle := len(amounts) / 2
	typical := amounts[middle]
	if len(amounts)%2 == 0 {
		typical = amounts[middle-1] + (typical-amounts[middle-1]+1)/2
	}
	return amounts, typical
}

func paycheckMonthlyAmount(typical int64, frequency string) (int64, bool) {
	var numerator, denominator int64
	switch frequency {
	case "weekly":
		numerator, denominator = 52, paycheckMonthsPerYear
	case "biweekly":
		numerator, denominator = 26, paycheckMonthsPerYear
	case "semimonthly":
		numerator, denominator = 2, 1
	case "monthly":
		numerator, denominator = 1, 1
	default:
		return 0, false
	}
	monthly := (typical*numerator + denominator/2) / denominator
	return monthly, monthly <= budget.MaxMoneyCents
}

func paycheckVariableAmount(amounts []int64, typical int64) bool {
	threshold := max(paycheckVariableMinimumCents, (typical*paycheckVariableBasisPoints+paycheckBasisPoints-1)/paycheckBasisPoints)
	return amounts[len(amounts)-1]-amounts[0] >= threshold
}

type paycheckFingerprintRecord struct {
	ID, ConnectionID, AccountID string
	Revision                    int64
	Date                        budget.CalendarDate
	AmountCents                 int64
}

func paycheckFingerprint(stream paycheckStream, suggestion budget.PaycheckSuggestion) (string, error) {
	facts := make([]paycheckFingerprintRecord, 0, len(stream.records))
	for _, record := range stream.records {
		facts = append(facts, paycheckFingerprintRecord{ID: record.ID, ConnectionID: record.ConnectionID,
			AccountID: record.AccountID, Revision: record.Revision, Date: record.Date, AmountCents: *record.AmountCents})
	}
	data, err := json.Marshal(struct {
		Account    budget.ActivityAccount
		Suggestion budget.PaycheckSuggestion
		Records    []paycheckFingerprintRecord
	}{stream.account, suggestion, facts})
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
