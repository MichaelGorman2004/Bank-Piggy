import assert from 'node:assert/strict';
import { test } from 'node:test';
import { actualMoneyFlowDescription, actualMoneyFlowState } from './actualMoneyFlowPresentation.ts';

/** Returns only the canonical inputs needed for recorded chart presentation. */
function response(patch = {}) {
  return {
    coverage: 'ready',
    totals: { incomeCents: -1000, spendingCents: -2001 },
    comparison: { bankAvailable: true, bankFlow: { totalCents: 2001 } },
    ...patch,
  };
}

test('unavailable bank history stays distinct from a recorded zero month', () => {
  for (const coverage of ['disabled', 'not_connected']) {
    const data = response({ coverage });
    assert.equal(actualMoneyFlowState(data), 'unavailable');
    assert.equal(
      actualMoneyFlowDescription(data),
      'Recorded bank income and spending are unavailable for this month.',
    );
  }
  assert.equal(
    actualMoneyFlowState(
      response({
        comparison: { bankAvailable: false, bankFlow: { totalCents: 0 } },
      }),
    ),
    'unavailable',
  );
  assert.equal(
    actualMoneyFlowState(
      response({
        comparison: { bankAvailable: true, bankFlow: { totalCents: 0 } },
      }),
    ),
    'empty',
  );
});

test('incomplete coverage uses recorded wording and preserves signed exact totals', () => {
  const data = response({ coverage: 'incomplete' });
  assert.equal(actualMoneyFlowState(data), 'ready');
  const text = actualMoneyFlowDescription(data);
  assert.ok(text.includes('Recorded income -$10.00.'));
  assert.ok(text.includes('Recorded spending -$20.01.'));
  assert.ok(text.includes('selected posted bank activity'));
  assert.ok(text.includes('not an account balance or proof of borrowing'));
});
