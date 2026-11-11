import assert from 'node:assert/strict';
import { test } from 'node:test';
import { format } from 'echarts/core';
import { moneyFlowTooltip } from './moneyFlowTooltip.ts';

/** Creates a graph with a long source label and its exact amount. */
function graphFixture(label) {
  return {
    nodes: [
      { name: 'member-id', label: { formatter: `${label}\n$1,234.56` } },
      { name: 'plan-hub', label: { formatter: '' } },
    ],
    links: [{ source: 'member-id', target: 'plan-hub', value: 123456 }],
  };
}

test('tooltips preserve long Unicode names and exact cents within the mobile text area', () => {
  const label = 'Alexandria👩‍👩‍👧‍👦Café東京MontgomeryAnderson'.repeat(2);
  const option = moneyFlowTooltip(graphFixture(label), 1);
  assert.equal(option.renderMode, 'richText');
  assert.equal(option.confine, true);
  const text = option.formatter({ name: 'member-id', value: 123456, dataType: 'node' });
  const lines = text.split('\n');
  assert.equal(lines.pop(), '$1,234.56');
  assert.equal(lines.join(''), label);
  for (const line of lines) assert.ok(format.getTextRect(line).width <= 222);
  assert.ok(text.includes('👩‍👩‍👧‍👦'));
  assert.ok(text.includes('é'));
});

test('wrapped links keep both visible labels and the exact edge amount', () => {
  const label = 'WideUnbrokenMemberName'.repeat(3);
  const option = moneyFlowTooltip(graphFixture(label), 1);
  const text = option.formatter({ dataType: 'edge', dataIndex: 0 });
  const lines = text.split('\n');
  assert.equal(lines.pop(), '$1,234.56');
  assert.equal(lines.join(''), `${label} → Your plan`);
  for (const line of lines) assert.ok(format.getTextRect(line).width <= 222);
});
