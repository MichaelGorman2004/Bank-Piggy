import assert from 'node:assert/strict';
import { test } from 'node:test';
import { actualMoneyFlowOption } from './actualMoneyFlowOption.ts';

/** Reads both sides of the chart without interpreting financial amounts. */
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

test('aggregate flow preserves exact cents without category or purchase rows', () => {
  const data = {
    inflows: [{ id: 'income', name: 'Recorded income', amountCents: 100001 }],
    outflows: [
      { id: 'spending', name: 'Spending', amountCents: 90000 },
      { id: 'surplus', name: 'Recorded surplus', amountCents: 10001 },
    ],
    totalCents: 100001,
  };
  const original = structuredClone(data);
  const chart = actualMoneyFlowOption(data, (token) => token).series[0];
  assert.deepEqual(chartTotals(chart), { inflows: 100001, outflows: 100001 });
  assert.deepEqual(chart.links, [
    { source: 'actual-inflow-income', target: 'actual-bank-hub', value: 100001 },
    { source: 'actual-bank-hub', target: 'actual-outflow-spending', value: 90000 },
    { source: 'actual-bank-hub', target: 'actual-outflow-surplus', value: 10001 },
  ]);
  assert.equal(
    chart.data.find((node) => node.name === 'actual-bank-hub').label.formatter,
    'Recorded bank flow\n$1,000.01',
  );
  assert.ok(chart.links.every((link) => link.value > 0));
  assert.deepEqual(data, original);
});

test('net refunds enter the chart while income reversals leave it', () => {
  const data = {
    inflows: [{ id: 'net-refunds', name: 'Net refunds', amountCents: 2001 }],
    outflows: [
      { id: 'income-reversals', name: 'Income reversals', amountCents: 1000 },
      { id: 'surplus', name: 'Recorded surplus', amountCents: 1001 },
    ],
    totalCents: 2001,
  };
  const chart = actualMoneyFlowOption(data, (token) => token).series[0];
  assert.deepEqual(chartTotals(chart), { inflows: 2001, outflows: 2001 });
  assert.equal(chart.links[0].source, 'actual-inflow-net-refunds');
  assert.equal(chart.links[1].target, 'actual-outflow-income-reversals');
  assert.equal(
    chart.data.find((node) => node.name.endsWith('income-reversals')).itemStyle.color,
    'danger',
  );
});

test('spending above recorded inflow retains the explicit gap with no inferred borrowing', () => {
  const data = {
    inflows: [{ id: 'inflow-gap', name: 'Spending above recorded inflow', amountCents: 501 }],
    outflows: [{ id: 'spending', name: 'Spending', amountCents: 501 }],
    totalCents: 501,
  };
  const chart = actualMoneyFlowOption(data, (token) => token).series[0];
  assert.deepEqual(chartTotals(chart), { inflows: 501, outflows: 501 });
  assert.ok(
    chart.data.some((node) => node.label.formatter.startsWith('Spending above recorded inflow')),
  );
  assert.ok(chart.links.every((link) => link.value > 0));
});

test('a recorded zero month has no invented bank flow', () => {
  const chart = actualMoneyFlowOption(
    { inflows: [], outflows: [], totalCents: 0 },
    (token) => token,
  ).series[0];
  assert.deepEqual(chart.data, []);
  assert.deepEqual(chart.links, []);
});
