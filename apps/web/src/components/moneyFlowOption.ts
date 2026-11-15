import type { HouseholdResponse, Summary } from '@bank-piggy/contracts';
import type { EChartsOption } from 'echarts';
import {
  flowCategoryCapacity,
  flowHub,
  flowNode,
  moneyFlowChartOption,
  type FlowColorReader,
  type FlowGraph,
  type FlowNode,
} from './moneyFlowChart.ts';
import { HOUSEHOLD_SCOPE, expenseAmount } from '../features/plan/format.ts';
import { compactFlowCategories } from './moneyFlowCategories.ts';

export interface MoneyFlowProps {
  readonly data: HouseholdResponse;
  readonly summary: Summary;
  readonly scope: string;
  readonly factor: number;
}
const HUB = 'plan-hub';
const FLOW_SERIES_ID = 'planned-money-flow';

/**
 * Names income separately from a plan total that also contains a shortfall source.
 * @param props Canonical selected summary and period scale.
 * @param graph Sources already added to the graph.
 * @param color Chart color token reader.
 * @returns Central bar with a complete amount above its geometry.
 */
function incomeHub(props: MoneyFlowProps, graph: FlowGraph, color: FlowColorReader): FlowNode {
  const shortfall = props.summary.monthlyRemainingCents < 0;
  const amount = shortfall
    ? graph.links.reduce((sum, link) => sum + (link.target === HUB ? link.value : 0), 0)
    : props.summary.monthlyTakeHomeCents;
  const label = shortfall
    ? 'Plan total'
    : props.scope === HOUSEHOLD_SCOPE && props.data.household.members.length > 1
      ? 'Combined income'
      : 'Income';
  return flowHub({ id: HUB, label }, amount, {
    factor: props.factor,
    color: color('taupe'),
  });
}

/**
 * Adds a terminal allocation with its exact cents and display label.
 * @param graph Mutable chart graph.
 * @param node Internal node identifier and visible label.
 * @param amount Amount in integer cents.
 */
function addAllocation(graph: FlowGraph, node: FlowNode, amount: number) {
  if (amount <= 0) return;
  graph.nodes.push(node);
  graph.links.push({ source: HUB, target: node.name, value: amount });
}

/**
 * Builds income sources, including an explicit funding gap when the plan is overallocated.
 * @param props Component values and event handlers.
 * @param graph Mutable chart graph.
 * @param color Chart color token reader.
 */
function addSources(props: MoneyFlowProps, graph: FlowGraph, color: FlowColorReader) {
  props.data.household.members
    .filter((member) => props.scope === HOUSEHOLD_SCOPE || member.id === props.scope)
    .forEach((member, index) => {
      const amount = props.data.summary.members[member.id].monthlyTakeHomeCents;
      if (amount <= 0) return;
      graph.nodes.push(
        flowNode({ id: member.id, label: member.name }, amount, {
          factor: props.factor,
          color: color(index === 0 ? 'orange' : 'gold'),
        }),
      );
      graph.links.push({ source: member.id, target: HUB, value: amount });
    });
  if (props.summary.monthlyRemainingCents >= 0) return;
  const shortfall = -props.summary.monthlyRemainingCents;
  graph.nodes.push(
    flowNode({ id: 'plan-shortfall', label: 'Plan shortfall' }, shortfall, {
      factor: props.factor,
      color: color('danger'),
    }),
  );
  graph.links.push({ source: 'plan-shortfall', target: HUB, value: shortfall });
}

/**
 * Groups exact server-owned expense shares by category for a readable flow.
 * @param props Component values and event handlers.
 * @param graph Mutable chart graph.
 * @param color Chart color token reader.
 * @param capacity Available expense category rows.
 */
function addExpenses(
  props: MoneyFlowProps,
  graph: FlowGraph,
  color: FlowColorReader,
  capacity: number,
) {
  const categories = new Map<string, number>();
  props.data.household.expenses.forEach((expense) => {
    const amount = expenseAmount(expense, props.data, props.scope);
    if (amount > 0)
      categories.set(expense.category, (categories.get(expense.category) ?? 0) + amount);
  });
  const shades = ['orange', 'gold', 'taupe', 'pink'];
  const rows = compactFlowCategories(
    [...categories].map(([label, amount]) => ({ id: `category-${label}`, label, amount })),
    capacity,
  );
  rows.forEach(({ id, label, amount }, index) => {
    addAllocation(
      graph,
      flowNode({ id, label }, amount, {
        factor: props.factor,
        color: color(shades[index % shades.length] ?? 'taupe'),
      }),
      amount,
    );
  });
}

/**
 * Creates a Sankey option with one balanced flow for the active scope.
 * @param props Component values and event handlers.
 * @param color Chart color token reader.
 * @param height Measured chart height in CSS pixels.
 * @returns The computed result.
 */
export function moneyFlowOption(
  props: MoneyFlowProps,
  color: FlowColorReader,
  height: number,
): EChartsOption {
  const graph: FlowGraph = { nodes: [], links: [] };
  addSources(props, graph, color);
  graph.nodes.push(incomeHub(props, graph, color));
  addExpenses(props, graph, color, categoryCapacity(props, height));
  addAllocation(
    graph,
    flowNode({ id: 'plan-goals', label: 'Goals' }, props.summary.monthlyGoalsCents, {
      factor: props.factor,
      color: color('sage'),
    }),
    props.summary.monthlyGoalsCents,
  );
  addAllocation(
    graph,
    flowNode({ id: 'plan-remaining', label: 'Left to plan' }, props.summary.monthlyRemainingCents, {
      factor: props.factor,
      color: color('walnut-soft'),
    }),
    props.summary.monthlyRemainingCents,
  );
  return moneyFlowChartOption(graph, color, props.factor, FLOW_SERIES_ID);
}

/**
 * Reserves legible rows for goals and the remaining amount before expense categories.
 * @param props Current chart scope and canonical plan.
 * @param height Measured chart height in CSS pixels.
 * @returns Available expense category rows, with at least one combined row.
 */
function categoryCapacity(props: MoneyFlowProps, height: number): number {
  const goalCount = props.summary.monthlyGoalsCents > 0 ? 1 : 0;
  const remainingCount = props.summary.monthlyRemainingCents > 0 ? 1 : 0;
  return flowCategoryCapacity(height, goalCount + remainingCount);
}
