import assert from 'node:assert/strict';
import { test } from 'node:test';
import { compactFlowCategories } from './moneyFlowCategories.ts';

test('compact flows retain every cent while keeping the largest costs visible', () => {
  const categories = [
    { id: 'food', label: 'Food', amount: 65001 },
    { id: 'rent', label: 'Housing', amount: 300000 },
    { id: 'bills', label: 'Bills', amount: 32001 },
    { id: 'travel', label: 'Transport', amount: 45000 },
  ];
  const original = structuredClone(categories);
  const rows = compactFlowCategories(categories, 3);
  assert.deepEqual(
    rows.map((row) => row.label),
    ['Housing', 'Food', 'Other costs (2)'],
  );
  assert.equal(rows.at(-1).amount, 77001);
  assert.equal(
    rows.reduce((sum, row) => sum + row.amount, 0),
    442002,
  );
  assert.deepEqual(categories, original);
  assert.deepEqual(compactFlowCategories(categories, 4), categories);
});

test('the smallest chart still represents all expenses and empty input stays empty', () => {
  assert.deepEqual(compactFlowCategories([], 1), []);
  const rows = compactFlowCategories(
    [
      { id: 'one', label: 'One', amount: 1 },
      { id: 'two', label: 'Two', amount: 2 },
    ],
    0,
  );
  assert.equal(rows.length, 1);
  assert.equal(rows[0].amount, 3);
});
