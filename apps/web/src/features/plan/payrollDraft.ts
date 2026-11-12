import type { Member } from '@bank-piggy/contracts';
import { moneyInput, parseMoney } from './format.ts';

type Payroll = NonNullable<Member['payroll']>;
export interface MatchTierDraft {
  readonly upTo: string;
  readonly rate: string;
}
export interface PayrollDraft {
  readonly enabled: boolean;
  readonly traditional: string;
  readonly roth: string;
  readonly method: Payroll['matchMethod'];
  readonly tiers: readonly MatchTierDraft[];
  readonly employerMonthly: string;
}
type PayrollResult =
  | { readonly payroll?: Payroll; readonly error?: never }
  | { readonly payroll?: never; readonly error: string };
const PERCENT_BPS = 10_000;
const MAX_MATCH_RATE_BPS = 20_000;
export const MAX_MATCH_TIERS = 5;

/**
 * Retains entered percentages separately from the canonical basis-point amounts.
 * @param payroll Saved payroll settings, when recorded.
 * @returns Editable payroll fields.
 */
export function payrollDraft(payroll?: Payroll): PayrollDraft {
  const value = payroll ?? {
    traditional401kBps: 0,
    roth401kBps: 0,
    matchMethod: 'none',
    matchTiers: [
      { upToContributionBps: 300, matchRateBps: 10_000 },
      { upToContributionBps: 500, matchRateBps: 5000 },
    ],
    employerMonthlyCents: 0,
  };
  return {
    enabled: Boolean(payroll),
    traditional: moneyInput(value.traditional401kBps),
    roth: moneyInput(value.roth401kBps),
    method: value.matchMethod,
    tiers: value.matchTiers.map((tier) => ({
      upTo: moneyInput(tier.upToContributionBps),
      rate: moneyInput(tier.matchRateBps),
    })),
    employerMonthly: moneyInput(value.employerMonthlyCents),
  };
}

/**
 * Converts cumulative match caps while rejecting overlapping or unordered tiers.
 * @param drafts Editable tier percentages, ordered by contribution cap.
 * @returns Validated canonical tiers, or null for invalid input.
 */
function matchTiers(drafts: readonly MatchTierDraft[]): Payroll['matchTiers'] | null {
  if (drafts.length === 0 || drafts.length > MAX_MATCH_TIERS) return null;
  const tiers: Payroll['matchTiers'] = [];
  let previousCap = 0;
  for (const draft of drafts) {
    const upToContributionBps = parseMoney(draft.upTo);
    const matchRateBps = parseMoney(draft.rate);
    if (upToContributionBps === null || matchRateBps === null) return null;
    if (upToContributionBps <= previousCap || upToContributionBps > PERCENT_BPS) return null;
    if (matchRateBps > MAX_MATCH_RATE_BPS) return null;
    tiers.push({ upToContributionBps, matchRateBps });
    previousCap = upToContributionBps;
  }
  return tiers;
}

/**
 * Validates payroll estimates without changing entered take-home pay.
 * @param draft Editable percentages and employer-match settings.
 * @returns Optional canonical settings or one field error.
 */
export function payrollValue(draft: PayrollDraft): PayrollResult {
  if (!draft.enabled) return {};
  const traditional401kBps = parseMoney(draft.traditional);
  const roth401kBps = parseMoney(draft.roth);
  if (traditional401kBps === null || roth401kBps === null)
    return { error: 'Enter 401(k) percentages with up to two decimal places.' };
  if (traditional401kBps + roth401kBps > PERCENT_BPS)
    return { error: 'Traditional and Roth 401(k) percentages must total 100% or less.' };
  const tiers = draft.method === 'tiers' ? matchTiers(draft.tiers) : [];
  if (!tiers) return { error: 'Enter match tiers with increasing salary caps from 0 to 100%.' };
  const fixed = draft.method === 'fixed' ? parseMoney(draft.employerMonthly) : 0;
  if (fixed === null || fixed > 100_000_000_000)
    return { error: 'Enter a valid monthly employer match amount.' };
  return {
    payroll: {
      traditional401kBps,
      roth401kBps,
      matchMethod: draft.method,
      matchTiers: tiers,
      employerMonthlyCents: fixed,
    },
  };
}
