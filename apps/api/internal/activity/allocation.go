package activity

import "bankpiggy/api/internal/budget"

func recordOwnership(record budget.ActivityRecord) string {
	if record.Allocation != nil {
		return record.Allocation.Ownership
	}
	return record.Ownership
}

// allocateRecord keeps raw payer facts separate from the selected member's cost share.
func allocateRecord(record budget.ActivityRecord, scope string, plan budget.Household) (budget.ActivityRecord, error) {
	record.ScopeAmountCents = nil
	if record.AmountCents == nil {
		return record, nil
	}
	amount := *record.AmountCents
	if !usableCents(record.AmountCents) {
		return record, ErrInvalid
	}
	if record.Allocation == nil {
		if record.ExpenseID != nil || record.GoalID != nil {
			return record, ErrInvalid
		}
		amount = scopedCents(amount, record.Ownership, scope, plan)
		record.ScopeAmountCents = &amount
		return record, nil
	}
	if !validAllocation(record) {
		return record, ErrInvalid
	}
	if scope == plan.Members[0].ID {
		amount = record.Allocation.FirstMemberCents
	} else if scope != "household" {
		amount -= record.Allocation.FirstMemberCents
	}
	record.ScopeAmountCents = &amount
	return record, nil
}

func validAllocation(record budget.ActivityRecord) bool {
	allocation := record.Allocation
	amount, first := *record.AmountCents, allocation.FirstMemberCents
	if (record.ExpenseID == nil) == (record.GoalID == nil) || first < min(0, amount) || first > max(0, amount) {
		return false
	}
	switch allocation.Ownership {
	case "member-1":
		return first == amount
	case "member-2":
		return first == 0
	case "shared":
		return true
	default:
		return false
	}
}
