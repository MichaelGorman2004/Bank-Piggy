import type { Member } from '@bank-piggy/contracts';

export interface PayScheduleDraft {
  readonly anchorDate: string;
  readonly firstDay: string;
  readonly secondDay: string;
  readonly monthEnd: boolean;
}
type ScheduleResult =
  | { readonly paySchedule?: Member['paySchedule']; readonly error?: never }
  | { readonly error: string; readonly paySchedule?: never };

/**
 * Copies the saved pay-date anchor without guessing dates for an existing plan.
 * @param member Member whose pay dates are being edited.
 * @returns Editable pay-date fields.
 */
export function payScheduleDraft(member: Member): PayScheduleDraft {
  return {
    anchorDate: member.paySchedule?.anchorDate ?? '',
    monthEnd: member.paySchedule?.monthEnd ?? false,
    firstDay: String(member.paySchedule?.semiMonthlyDays[0] ?? 1),
    secondDay: String(member.paySchedule?.semiMonthlyDays[1] ?? 15),
  };
}

/**
 * Checks an actual calendar date without treating it as a UTC timestamp.
 * @param text Date in YYYY-MM-DD form.
 * @returns Whether the date exists and uses the supported four-digit year range.
 */
export function validDate(text: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(text)) return false;
  const date = new Date(`${text}T12:00:00`);
  return (
    !Number.isNaN(date.valueOf()) &&
    localDate(date) === text &&
    date.getFullYear() >= 2000 &&
    date.getFullYear() <= 2100
  );
}

/**
 * Formats a local date so the current pay period follows the user's calendar.
 * @param date Local date to format.
 * @returns A YYYY-MM-DD date without UTC conversion.
 */
export function localDate(date: Date): string {
  return `${String(date.getFullYear()).padStart(4, '0')}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

/**
 * Validates an optional schedule and keeps irregular pay free of inferred dates.
 * @param draft Editable anchor and twice-monthly dates.
 * @param frequency Canonical pay frequency.
 * @returns A canonical schedule or one actionable error.
 */
export function payScheduleValue(
  draft: PayScheduleDraft,
  frequency: Member['payFrequency'],
): ScheduleResult {
  if (frequency === 'irregular' || !draft.anchorDate) return {};
  if (!validDate(draft.anchorDate)) return { error: 'Choose a valid pay date.' };
  if (frequency === 'monthly' && draft.monthEnd) {
    const anchor = new Date(`${draft.anchorDate}T12:00:00`);
    if (anchor.getDate() !== monthLastDay(anchor))
      return { error: 'Choose a pay date that is the last day of its month.' };
    return { paySchedule: { anchorDate: draft.anchorDate, semiMonthlyDays: [], monthEnd: true } };
  }
  if (frequency === 'semimonthly') return semiMonthlyValue(draft);
  return { paySchedule: { anchorDate: draft.anchorDate, semiMonthlyDays: [] } };
}

/**
 * Reads a month's last date without UTC conversion.
 * @param date Date within the calendar month.
 * @returns The last calendar day of that month.
 */
function monthLastDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate();
}

/**
 * Validates two monthly pay days and requires the anchor to be one of them.
 * @param draft Editable anchor and monthly dates.
 * @returns Canonical schedule or one field error.
 */
function semiMonthlyValue(draft: PayScheduleDraft): ScheduleResult {
  const days = [draft.firstDay, draft.secondDay]
    .map(Number)
    .sort((first, second) => first - second);
  if (days.some((day) => !Number.isInteger(day) || day < 1 || day > 31) || days[0] === days[1])
    return { error: 'Choose two different monthly pay days from 1 through 31.' };
  const first = days[0];
  if (first >= 28)
    return {
      error:
        'Choose a first pay day before the 28th so both pay dates remain separate in February.',
    };
  const anchor = new Date(`${draft.anchorDate}T12:00:00`);
  if (!days.some((day) => Math.min(day, monthLastDay(anchor)) === anchor.getDate()))
    return { error: 'Choose a pay date that matches one of the two monthly pay days.' };
  return { paySchedule: { anchorDate: draft.anchorDate, semiMonthlyDays: days } };
}
