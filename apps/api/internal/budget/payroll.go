package budget

import "errors"

const basisPoints int64 = 10000
const monthsPerYear int64 = 12
const maxMatchRateBps int64 = 20000
const maxMatchTiers = 5

func validContributionBps(value int64) bool {
	return value >= 0 && value <= basisPoints
}

func validatePayroll(payroll *Payroll) error {
	if payroll == nil {
		return nil
	}
	if !validContributionBps(payroll.Traditional401KBps) || !validContributionBps(payroll.Roth401KBps) || payroll.Traditional401KBps+payroll.Roth401KBps > basisPoints {
		return errors.New("combined employee payroll contributions must be between zero and 10000 basis points")
	}
	if !validMoney(payroll.EmployerMonthlyCents) || payroll.MatchTiers == nil {
		return errors.New("employer contribution must be valid whole cents and match tiers must be an array")
	}
	return validateEmployerMatch(payroll)
}

func validateEmployerMatch(payroll *Payroll) error {
	switch payroll.MatchMethod {
	case "none":
		if len(payroll.MatchTiers) != 0 || payroll.EmployerMonthlyCents != 0 {
			return errors.New("no employer match cannot include tiers or a fixed contribution")
		}
	case "fixed":
		if len(payroll.MatchTiers) != 0 {
			return errors.New("a fixed employer contribution cannot include match tiers")
		}
	case "tiers":
		return validateMatchTiers(payroll)
	default:
		return errors.New("invalid employer match method")
	}
	return nil
}

func validateMatchTiers(payroll *Payroll) error {
	if len(payroll.MatchTiers) < 1 || len(payroll.MatchTiers) > maxMatchTiers || payroll.EmployerMonthlyCents != 0 {
		return errors.New("tiered match needs one to five tiers and no fixed employer contribution")
	}
	var previous int64
	for _, tier := range payroll.MatchTiers {
		if tier.UpToContributionBps <= previous || tier.UpToContributionBps > basisPoints || tier.MatchRateBps < 0 || tier.MatchRateBps > maxMatchRateBps {
			return errors.New("match tiers need increasing thresholds and rates between zero and 20000 basis points")
		}
		previous = tier.UpToContributionBps
	}
	return nil
}

func employerMonthlyEstimate(member Member) int64 {
	payroll := member.Payroll
	if payroll.MatchMethod == "fixed" {
		return payroll.EmployerMonthlyCents
	}
	if payroll.MatchMethod != "tiers" {
		return 0
	}
	contribution := payroll.Traditional401KBps + payroll.Roth401KBps
	var previous, weightedRate int64
	for _, tier := range payroll.MatchTiers {
		slice := max(0, min(contribution, tier.UpToContributionBps)-previous)
		weightedRate += slice * tier.MatchRateBps
		previous = tier.UpToContributionBps
	}
	return proportional(member.AnnualBaseSalaryCents, weightedRate, monthsPerYear*basisPoints*basisPoints)
}

// payrollSummaries estimates contributions from gross base salary. These amounts
// never change entered take-home pay, allocations, or account balances.
func payrollSummaries(members []Member) []PayrollSummary {
	result := make([]PayrollSummary, 0, len(members))
	for _, member := range members {
		summary := PayrollSummary{MemberID: member.ID}
		if payroll := member.Payroll; payroll != nil {
			employeeMonthly := proportional(member.AnnualBaseSalaryCents, payroll.Traditional401KBps+payroll.Roth401KBps, monthsPerYear*basisPoints)
			summary.Traditional401KMonthlyCents = proportional(member.AnnualBaseSalaryCents, payroll.Traditional401KBps, monthsPerYear*basisPoints)
			// Assign Roth the remainder so employee contributions keep every cent.
			summary.Roth401KMonthlyCents = employeeMonthly - summary.Traditional401KMonthlyCents
			summary.EmployerMonthlyCents = employerMonthlyEstimate(member)
		}
		result = append(result, summary)
	}
	return result
}
