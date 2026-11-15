/** Gives request tests deterministic control of completion order. */
export function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((accept, decline) => {
    resolve = accept;
    reject = decline;
  });
  return { promise, resolve, reject };
}

/** Returns a recorded monthly response without embedding sample data in the app. */
export function activityResponse(patch = {}) {
  return {
    month: '2026-11',
    scope: 'household',
    planRevision: 3,
    environment: 'sandbox',
    coverage: 'not_connected',
    notice: 'No recorded history.',
    totals: {
      incomeCents: 0,
      spendingCents: 0,
      investmentDepositsCents: 0,
      investmentWithdrawalsCents: 0,
      netInvestmentContributionsCents: 0,
      investmentIncomeCents: 0,
      investmentFeesCents: 0,
    },
    suggestions: [],
    sources: [],
    comparison: activityComparison(),
    ...patch,
  };
}

/** Creates authoritative plan comparisons and availability without sample runtime data. */
export function activityComparison(patch = {}) {
  return {
    bankAvailable: false,
    investmentAvailable: false,
    expenseBudgetCents: 0,
    expenseRemainingCents: 0,
    investmentGoalBudgetCents: 0,
    payrollBudgetCents: 0,
    investmentBudgetCents: 0,
    investmentRemainingCents: 0,
    bankFlow: { inflows: [], outflows: [], totalCents: 0 },
    ...patch,
  };
}

/** Records visible state separately from the mutable request owner. */
export function activityOwner(load = async () => activityResponse()) {
  const state = {
    data: null,
    loading: false,
    busy: false,
    error: null,
    feedback: null,
    totalsStale: false,
    changes: 0,
  };
  const owner = {
    mounted: true,
    generation: 0,
    locked: false,
    query: { month: '2026-11', scope: 'household' },
    data: null,
    load,
    setData: (value) => {
      state.data = value;
      state.changes += 1;
    },
    setLoading: (value) => {
      state.loading = value;
      state.changes += 1;
    },
    setBusy: (value) => {
      state.busy = value;
      state.changes += 1;
    },
    setError: (value) => {
      state.error = value;
      state.changes += 1;
    },
    setTotalsStale: (value) => {
      state.totalsStale = value;
      state.changes += 1;
    },
    setFeedback: (value) => {
      state.feedback = value;
      state.changes += 1;
    },
  };
  return { owner, state };
}

/** Creates one selected history source with explicit account coverage. */
export function activitySource(patch = {}) {
  return {
    connectionId: 'connection-bank',
    institutionName: 'Test institution',
    kind: 'depository',
    enabled: true,
    allowed: true,
    canManage: true,
    revision: 1,
    needsConsent: false,
    state: 'ready',
    lastSyncedAt: '2026-11-13T12:00:00Z',
    historyStart: '2026-08-01',
    historyEnd: '2026-11-13',
    errorCode: null,
    accounts: [
      {
        accountId: 'account-bank',
        name: 'Checking',
        providerType: 'depository',
        currency: 'USD',
        available: true,
        enabled: true,
        ownership: 'member-1',
      },
    ],
    ...patch,
  };
}
