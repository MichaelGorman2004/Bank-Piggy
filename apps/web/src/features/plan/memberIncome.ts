import type { Member } from '@bank-piggy/contracts';
import type { MemberDraft } from './planDraft.ts';
import { parseMoney } from './format.ts';
import { payrollValue } from './payrollDraft.ts';
import { payScheduleValue } from './payScheduleDraft.ts';

type IncomeFields = Pick<
  Member,
  | 'name'
  | 'annualBaseSalaryCents'
  | 'monthlyTakeHomeCents'
  | 'payFrequency'
  | 'payroll'
  | 'paySchedule'
>;
type IncomeResult =
  | { readonly fields: IncomeFields; readonly error?: never }
  | { readonly error: string; readonly fields?: never };

/**
 * Validates income, payroll estimates, and schedule inputs before plan persistence.
 * @param draft Editable member income and payroll settings.
 * @returns Canonical income fields or one actionable validation error.
 */
export function memberIncomeValue(draft: MemberDraft): IncomeResult {
  const annualBaseSalaryCents = parseMoney(draft.salary);
  const monthlyTakeHomeCents = parseMoney(draft.takeHome);
  if (annualBaseSalaryCents === null || monthlyTakeHomeCents === null || !draft.name.trim())
    return {
      error:
        'Check each member’s name, take-home pay, and salary. Use nonnegative amounts with up to two decimal places.',
    };
  const payroll = payrollValue(draft.payroll);
  if (payroll.error) return { error: payroll.error };
  const schedule = payScheduleValue(draft.paySchedule, draft.frequency);
  if (schedule.error) return { error: schedule.error };
  return {
    fields: {
      name: draft.name.trim(),
      annualBaseSalaryCents,
      monthlyTakeHomeCents,
      payFrequency: draft.frequency,
      ...(payroll.payroll ? { payroll: payroll.payroll } : {}),
      ...(schedule.paySchedule ? { paySchedule: schedule.paySchedule } : {}),
    },
  };
}
