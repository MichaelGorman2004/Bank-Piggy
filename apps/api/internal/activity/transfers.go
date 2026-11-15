package activity

import (
	"math"
	"sort"
	"time"

	"bankpiggy/api/internal/budget"
)

const transferWindowDays = 7
const maxTransferCandidates = 12

type transferKey struct {
	amount    int64
	direction budget.ActivityCategory
}

// Reconcile identifies possible investment transfers across month boundaries without asserting identity.
// Explicit pairs remain visible; ambiguous unreviewed funding is excluded until reviewed.
func Reconcile(records []budget.ActivityRecord) []budget.ActivityRecord {
	result := make([]budget.ActivityRecord, len(records))
	byID := make(map[string]budget.ActivityRecord, len(records))
	groups := map[transferKey]map[string][]budget.ActivityRecord{}
	for index, record := range records {
		result[index] = Classify(record)
		byID[record.ID] = result[index]
		if transferEligible(result[index]) {
			addTransferGroup(groups, result[index])
		}
	}
	for _, accounts := range groups {
		for account := range accounts {
			sort.Slice(accounts[account], func(i, j int) bool { return accounts[account][i].Date < accounts[account][j].Date })
		}
	}
	for index := range result {
		addTransferCandidates(&result[index], byID, groups)
	}
	return result
}

func transferEligible(record budget.ActivityRecord) bool {
	if record.Kind != "investment" || record.AmountCents == nil || *record.AmountCents == 0 || record.Pending || record.Cancelled || record.Currency == nil || *record.Currency != "USD" {
		return false
	}
	if record.IsReviewed && record.Category != "transfer" {
		return false
	}
	return transferDirection(record) != ""
}

func transferDirection(record budget.ActivityRecord) budget.ActivityCategory {
	category := investmentCategory(record)
	if category == "contribution" || category == "withdrawal" {
		return category
	}
	return ""
}

func addTransferGroup(groups map[transferKey]map[string][]budget.ActivityRecord, record budget.ActivityRecord) {
	key := transferKey{absolute(*record.AmountCents), transferDirection(record)}
	if groups[key] == nil {
		groups[key] = map[string][]budget.ActivityRecord{}
	}
	groups[key][record.AccountID] = append(groups[key][record.AccountID], record)
}

func addTransferCandidates(record *budget.ActivityRecord, byID map[string]budget.ActivityRecord, groups map[transferKey]map[string][]budget.ActivityRecord) {
	if record.TransferPeerID != nil {
		appendSavedTransferPeer(record, byID)
		return
	}
	if !transferEligible(*record) {
		return
	}
	opposite := budget.ActivityCategory("withdrawal")
	if transferDirection(*record) == "withdrawal" {
		opposite = "contribution"
	}
	key := transferKey{absolute(*record.AmountCents), opposite}
	accountIDs := make([]string, 0, len(groups[key]))
	for account := range groups[key] {
		accountIDs = append(accountIDs, account)
	}
	sort.Strings(accountIDs)
	for _, account := range accountIDs {
		if account != record.AccountID {
			appendNearbyCandidates(record, groups[key][account])
		}
	}
	sort.Slice(record.TransferCandidates, func(i, j int) bool {
		return record.TransferCandidates[i].RecordID < record.TransferCandidates[j].RecordID
	})
	if len(record.TransferCandidates) > 0 && !record.IsReviewed {
		*record = reason(*record, "review", "This may be a transfer between investment accounts. Confirm it before counting new funding.")
	}
}

func appendSavedTransferPeer(record *budget.ActivityRecord, byID map[string]budget.ActivityRecord) {
	if peer, ok := byID[*record.TransferPeerID]; ok && peer.AmountCents != nil {
		record.TransferCandidates = append(record.TransferCandidates, transferCandidate(peer))
	}
}

func appendNearbyCandidates(record *budget.ActivityRecord, candidates []budget.ActivityRecord) {
	date, err := time.Parse("2006-01-02", string(record.Date))
	if err != nil {
		return
	}
	start, end := date.AddDate(0, 0, -transferWindowDays).Format("2006-01-02"), date.AddDate(0, 0, transferWindowDays).Format("2006-01-02")
	first := sort.Search(len(candidates), func(index int) bool { return string(candidates[index].Date) >= start })
	for _, candidate := range candidates[first:] {
		if string(candidate.Date) > end || len(record.TransferCandidates) >= maxTransferCandidates {
			break
		}
		if candidate.TransferPeerID == nil {
			record.TransferCandidates = append(record.TransferCandidates, transferCandidate(candidate))
		}
	}
}

func transferCandidate(record budget.ActivityRecord) budget.ActivityTransferCandidate {
	return budget.ActivityTransferCandidate{RecordID: record.ID, Revision: record.Revision, Date: record.Date, AccountName: record.AccountName, AmountCents: absolute(*record.AmountCents)}
}

// ValidTransferPair accepts opposite funding legs only; equal amount/date alone never confirms a pair.
func ValidTransferPair(first, second budget.ActivityRecord) bool {
	if first.ID == second.ID || first.AccountID == second.AccountID || first.AmountCents == nil || second.AmountCents == nil {
		return false
	}
	first.IsReviewed, second.IsReviewed = false, false
	first, second = Classify(first), Classify(second)
	return transferEligible(first) && transferEligible(second) && transferDatesMatch(first.Date, second.Date) && transferDirection(first) != transferDirection(second) && absolute(*first.AmountCents) == absolute(*second.AmountCents)
}

func transferDatesMatch(first, second budget.CalendarDate) bool {
	a, aErr := time.Parse(time.DateOnly, string(first))
	b, bErr := time.Parse(time.DateOnly, string(second))
	return aErr == nil && bErr == nil && math.Abs(a.Sub(b).Hours()) <= transferWindowDays*24
}
