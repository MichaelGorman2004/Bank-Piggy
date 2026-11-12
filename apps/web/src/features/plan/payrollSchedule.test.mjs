import assert from 'node:assert/strict';
import { test } from 'node:test';
import { payrollDraft, payrollValue } from './payrollDraft.ts';
import { payScheduleDraft, payScheduleValue, validDate, localDate } from './payScheduleDraft.ts';
import { memberDraft, convertPlan } from './planDraft.ts';
import { initialPlan } from './planReducer.ts';

const MEMBER = {
  id: 'member-1',
  name: 'Alex',
  annualBaseSalaryCents: 12000000,
  monthlyTakeHomeCents: 560001,
  debtCents: 0,
  accounts: [],
  payFrequency: 'monthly',
};

test('payroll settings retain exact percentages and never change entered take-home pay', () => {
  const payroll = {
    traditional401kBps: 425,
    roth401kBps: 275,
    matchMethod: 'tiers',
    matchTiers: [
      { upToContributionBps: 300, matchRateBps: 10000 },
      { upToContributionBps: 500, matchRateBps: 5000 },
    ],
    employerMonthlyCents: 0,
  };
  const member = { ...MEMBER, payroll };
  assert.deepEqual(payrollValue(payrollDraft(payroll)), { payroll });
  const household = {
    revision: 1,
    members: [member],
    expenses: [],
    goals: [],
    defaultSplitMethod: 'equal',
  };
  const draft = initialPlan(household);
  draft.members[0].payroll = { ...draft.members[0].payroll, traditional: '5.01' };
  const original = structuredClone(draft);
  const result = convertPlan(household, draft);
  assert.equal(result.household.members[0].payroll.traditional401kBps, 501);
  assert.equal(result.household.members[0].monthlyTakeHomeCents, 560001);
  assert.deepEqual(draft, original);
  assert.equal(memberDraft(member).payroll.roth, '2.75');
});

test('invalid payroll rates and overlapping match tiers retain editable input', () => {
  const draft = { ...payrollDraft(), enabled: true, method: 'tiers' };
  const invalid = [
    { ...draft, traditional: '99', roth: '1.01' },
    { ...draft, traditional: '1.001' },
    {
      ...draft,
      tiers: [
        { upTo: '3', rate: '100' },
        { upTo: '3', rate: '50' },
      ],
    },
    {
      ...draft,
      tiers: [
        { upTo: '5', rate: '100' },
        { upTo: '3', rate: '50' },
      ],
    },
    { ...draft, tiers: [{ upTo: '5', rate: '200.01' }] },
    { ...draft, tiers: [] },
  ];
  for (const value of invalid) {
    const original = structuredClone(value);
    assert.ok(payrollValue(value).error);
    assert.deepEqual(value, original);
  }
  const fixed = payrollValue({ ...draft, method: 'fixed', employerMonthly: '123.45' });
  assert.equal(fixed.payroll.employerMonthlyCents, 12345);
  assert.deepEqual(fixed.payroll.matchTiers, []);
  assert.equal(payrollValue({ ...draft, enabled: false }).payroll, undefined);
});

test('pay dates validate actual dates and retain explicit monthly-end intent', () => {
  assert.equal(validDate('2028-02-29'), true);
  assert.equal(validDate('2027-02-29'), false);
  assert.equal(validDate('1999-12-31'), false);
  assert.equal(validDate('2101-01-01'), false);
  assert.equal(localDate(new Date(2026, 10, 11, 23, 59)), '2026-11-11');
  const draft = { ...payScheduleDraft(MEMBER), anchorDate: '2027-02-28', monthEnd: true };
  assert.deepEqual(payScheduleValue(draft, 'monthly').paySchedule, {
    anchorDate: '2027-02-28',
    semiMonthlyDays: [],
    monthEnd: true,
  });
  assert.ok(payScheduleValue({ ...draft, anchorDate: '2027-02-27' }, 'monthly').error);
  assert.deepEqual(payScheduleValue(draft, 'weekly').paySchedule, {
    anchorDate: '2027-02-28',
    semiMonthlyDays: [],
  });
  assert.equal(payScheduleValue(draft, 'irregular').paySchedule, undefined);
});

test('twice-monthly dates reject February collisions and anchors outside the entered days', () => {
  const draft = {
    ...payScheduleDraft(MEMBER),
    anchorDate: '2027-02-28',
    firstDay: '15',
    secondDay: '31',
  };
  assert.deepEqual(payScheduleValue(draft, 'semimonthly').paySchedule, {
    anchorDate: '2027-02-28',
    semiMonthlyDays: [15, 31],
  });
  assert.ok(payScheduleValue({ ...draft, firstDay: '28' }, 'semimonthly').error);
  assert.ok(payScheduleValue({ ...draft, anchorDate: '2027-02-14' }, 'semimonthly').error);
  assert.ok(payScheduleValue({ ...draft, firstDay: '15', secondDay: '15' }, 'semimonthly').error);
  assert.equal(payScheduleValue({ ...draft, anchorDate: '' }, 'monthly').paySchedule, undefined);
});
