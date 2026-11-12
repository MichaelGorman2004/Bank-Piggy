import assert from 'node:assert/strict';
import { test } from 'node:test';
import { moneyFlowOption } from './moneyFlowOption.ts';

/** Creates canonical server shares for one-person and two-person chart inputs. */
function flowFixture(count) {
  const members = ['member-1', 'member-2'].slice(0, count).map((id) => ({ id, name: id }));
  const summary = {
    monthlyTakeHomeCents: count * 400001,
    monthlyGoalsCents: 2501,
    monthlyRemainingCents: count * 400001 - 100001 - 2501,
  };
  const data = {
    household: {
      members,
      expenses: [{ id: 'rent', category: 'Housing', ownerId: 'shared' }],
    },
    summary: {
      household: summary,
      members: Object.fromEntries(
        members.map((member) => [member.id, { monthlyTakeHomeCents: 400001 }]),
      ),
      expenseShares: [
        {
          expenseId: 'rent',
          firstMemberCents: count === 1 ? 100001 : 50001,
          secondMemberCents: count === 1 ? 0 : 50000,
        },
      ],
    },
  };
  return { data, summary, scope: 'household', factor: 1 };
}

test('a one-person chart uses one income source and retains every server-allocated cent', () => {
  const props = flowFixture(1);
  for (const scope of ['household', 'member-1']) {
    const chart = moneyFlowOption({ ...props, scope }, (token) => token, 600).series[0];
    const sources = chart.links.filter((link) => link.target === 'plan-hub');
    assert.deepEqual(sources, [{ source: 'member-1', target: 'plan-hub', value: 400001 }]);
    assert.equal(
      chart.data.find((node) => node.name === 'plan-hub').label.formatter,
      'Income\n$4,000.01',
    );
    assert.equal(
      chart.data.some((node) => node.name === 'member-2'),
      false,
    );
    assert.equal(chart.links.find((link) => link.target === 'category-Housing').value, 100001);
    assert.equal(
      chart.links
        .filter((link) => link.source === 'plan-hub')
        .reduce((sum, link) => sum + link.value, 0),
      400001,
    );
  }
});

test('a two-person chart retains its combined income label and both sources', () => {
  const props = flowFixture(2);
  const chart = moneyFlowOption(props, (token) => token, 600).series[0];
  assert.equal(
    chart.data.find((node) => node.name === 'plan-hub').label.formatter,
    'Combined income\n$8,000.02',
  );
  assert.deepEqual(
    chart.links.filter((link) => link.target === 'plan-hub').map((link) => link.source),
    ['member-1', 'member-2'],
  );
  assert.equal(chart.links.find((link) => link.target === 'category-Housing').value, 100001);
});
