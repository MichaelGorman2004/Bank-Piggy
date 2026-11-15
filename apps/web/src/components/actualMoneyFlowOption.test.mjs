import assert from 'node:assert/strict';
import { test } from 'node:test';
import { actualMoneyFlowOption } from './actualMoneyFlowOption.ts';

/** Returns one balanced recorded flow with explicit server-owned differences. */
function bankFlow() {
  return {
    inflows: [
      { id: 'income', name: 'Recorded income', amountCents: 100001 },
      { id: 'refund-Food', name: 'Refunds: Food', amountCents: 1001 },
    ],
    outflows: [
      { id: 'spending-Housing', name: 'Housing', amountCents: 90000 },
      { id: 'income-reversals', name: 'Income reversals', amountCents: 2000 },
      { id: 'surplus', name: 'Recorded surplus', amountCents: 9002 },
    ],
    totalCents: 101002,
  };
}

/** Reads both sides of a generated Sankey without interpreting any financial rows. */
function chartTotals(chart) {
  return {
    inflows: chart.links
      .filter((link) => link.target === 'actual-bank-hub')
      .reduce((sum, link) => sum + link.value, 0),
    outflows: chart.links
      .filter((link) => link.source === 'actual-bank-hub')
      .reduce((sum, link) => sum + link.value, 0),
  };
}

test('recorded flow preserves canonical cents, refund direction, and income reversals', () => {
  const data = bankFlow();
  const original = structuredClone(data);
  const chart = actualMoneyFlowOption(data, (token) => token, 600).series[0];
  assert.deepEqual(chartTotals(chart), { inflows: data.totalCents, outflows: data.totalCents });
  assert.deepEqual(
    chart.links.find((link) => link.source === 'actual-inflow-refund-Food'),
    { source: 'actual-inflow-refund-Food', target: 'actual-bank-hub', value: 1001 },
  );
  assert.equal(
    chart.links.find((link) => link.target === 'actual-outflow-income-reversals').value,
    2000,
  );
  assert.equal(
    chart.data.find((node) => node.name === 'actual-bank-hub').label.formatter,
    'Recorded bank flow\n$1,010.02',
  );
  assert.ok(chart.links.every((link) => link.value > 0));
  assert.deepEqual(data, original);
});

test('small charts compact categories but retain the explicit gap and income reversal', () => {
  const categories = Array.from({ length: 30 }, (_, index) => ({
    id: `spending-Category ${String(index)}`,
    name: `Category ${String(index)}`,
    amountCents: (index + 1) * 100 + 1,
  }));
  const totalCents = categories.reduce((sum, row) => sum + row.amountCents, 0) + 200;
  const data = {
    inflows: [
      { id: 'income', name: 'Recorded income', amountCents: 10000 },
      { id: 'inflow-gap', name: 'Spending above recorded inflow', amountCents: totalCents - 10000 },
    ],
    outflows: [
      ...categories,
      { id: 'income-reversals', name: 'Income reversals', amountCents: 200 },
    ],
    totalCents,
  };
  const chart = actualMoneyFlowOption(data, (token) => token, 320).series[0];
  assert.deepEqual(chartTotals(chart), { inflows: totalCents, outflows: totalCents });
  const labels = chart.data.map((node) => node.label.formatter);
  assert.ok(labels.some((label) => label.startsWith('Category 29\n')));
  assert.ok(labels.some((label) => label.startsWith('Other costs (')));
  assert.ok(labels.some((label) => label.startsWith('Spending above recorded inflow\n')));
  assert.ok(labels.some((label) => label.startsWith('Income reversals\n')));
  assert.ok(chart.links.filter((link) => link.source === 'actual-bank-hub').length <= 6);
});

test('refund compaction retains all cents and names the combined refund stream', () => {
  const refunds = Array.from({ length: 8 }, (_, index) => ({
    id: `refund-Category ${String(index)}`,
    name: `Refunds: Category ${String(index)}`,
    amountCents: index + 1,
  }));
  const data = {
    inflows: refunds,
    outflows: [{ id: 'surplus', name: 'Recorded surplus', amountCents: 36 }],
    totalCents: 36,
  };
  const chart = actualMoneyFlowOption(data, (token) => token, 220).series[0];
  assert.deepEqual(chartTotals(chart), { inflows: 36, outflows: 36 });
  assert.equal(
    chart.data.find((node) => node.name === 'actual-inflow-grouped-expenses').label.formatter,
    'Other refunds (6)\n$0.21',
  );
  assert.equal(
    chart.data.find((node) => node.name === 'actual-outflow-surplus').label.formatter,
    'Recorded surplus\n$0.36',
  );
});

test('recorded chart tooltips keep editable markup as text rather than rendered HTML', () => {
  const label = '<img src=x onerror=alert(1)> & <b>Refunds</b>';
  const data = {
    inflows: [{ id: 'income', name: label, amountCents: 12345 }],
    outflows: [{ id: 'surplus', name: 'Recorded surplus', amountCents: 12345 }],
    totalCents: 12345,
  };
  const option = actualMoneyFlowOption(data, (token) => token, 600);
  assert.equal(option.tooltip.renderMode, 'richText');
  const text = option.tooltip.formatter({
    name: 'actual-inflow-income',
    value: 12345,
    dataType: 'node',
  });
  assert.equal(text.split('\n').slice(0, -1).join(''), label);
  assert.ok(text.endsWith('$123.45'));
});

test('a recorded zero month has no invented bank-flow geometry', () => {
  const option = actualMoneyFlowOption(
    { inflows: [], outflows: [], totalCents: 0 },
    (token) => token,
    220,
  );
  assert.deepEqual(option.series[0].data, []);
  assert.deepEqual(option.series[0].links, []);
});
