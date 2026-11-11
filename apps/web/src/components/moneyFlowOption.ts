import type { HouseholdResponse, Summary } from '@bank-piggy/contracts';
import type { EChartsOption } from 'echarts';
import type { SankeySeriesOption } from 'echarts/charts';
import { moneyFlowTooltip } from './moneyFlowTooltip.ts';
import { HOUSEHOLD_SCOPE, expenseAmount, money } from '../features/plan/format.ts';
import { compactFlowCategories } from './moneyFlowCategories.ts';

export interface MoneyFlowProps {
  readonly data: HouseholdResponse;
  readonly summary: Summary;
  readonly scope: string;
  readonly factor: number;
}
type ColorReader = (name: string) => string;
interface FlowNode {
  readonly name: string;
  readonly itemStyle: { readonly color: string };
  readonly label: NonNullable<SankeySeriesOption['label']> & { readonly formatter: string };
}
interface FlowLink {
  readonly source: string;
  readonly target: string;
  readonly value: number;
}
export interface FlowGraph {
  readonly nodes: FlowNode[];
  readonly links: FlowLink[];
}
const HUB = 'plan-hub';
const FLOW_SERIES_ID = 'planned-money-flow';
const FLOW_TOP_PADDING = 80;
const FLOW_BOTTOM_PADDING = 22;
const FLOW_ROW_GAP = 36;

/**
 * Names income separately from a plan total that also contains a shortfall source.
 * @param props Canonical selected summary and period scale.
 * @param graph Sources already added to the graph.
 * @param color Chart color token reader.
 * @returns Central bar with a complete amount above its geometry.
 */
function incomeHub(props: MoneyFlowProps, graph: FlowGraph, color: ColorReader): FlowNode {
  const shortfall = props.summary.monthlyRemainingCents < 0;
  const amount = shortfall
    ? graph.links.reduce((sum, link) => sum + (link.target === HUB ? link.value : 0), 0)
    : props.summary.monthlyTakeHomeCents;
  const label = shortfall
    ? 'Plan total'
    : props.scope === HOUSEHOLD_SCOPE
      ? 'Combined income'
      : 'Income';
  const node = flowNode({ id: HUB, label }, amount, props.factor, color('taupe'));
  return {
    ...node,
    label: {
      ...node.label,
      position: 'top',
      align: 'center',
      distance: 24,
      width: 128,
      overflow: 'break',
      fontWeight: 600,
    },
  };
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
 * Builds labels from canonical amounts; the chart geometry always uses unscaled cents.
 * @param node Internal node identifier and visible label.
 * @param node.id Stable internal graph identifier.
 * @param node.label Editable display label.
 * @param amount Amount in integer cents.
 * @param factor Scale for the selected estimated period.
 * @param color Chart color token reader.
 * @returns The computed result.
 */
function flowNode(
  node: { readonly id: string; readonly label: string },
  amount: number,
  factor: number,
  color: string,
): FlowNode {
  return {
    name: node.id,
    itemStyle: { color },
    label: { formatter: `${node.label}\n${money(amount, factor)}` },
  };
}

/**
 * Builds income sources, including an explicit funding gap when the plan is overallocated.
 * @param props Component values and event handlers.
 * @param graph Mutable chart graph.
 * @param color Chart color token reader.
 */
function addSources(props: MoneyFlowProps, graph: FlowGraph, color: ColorReader) {
  props.data.household.members
    .filter((member) => props.scope === HOUSEHOLD_SCOPE || member.id === props.scope)
    .forEach((member, index) => {
      const amount = props.data.summary.members[member.id].monthlyTakeHomeCents;
      if (amount <= 0) return;
      graph.nodes.push(
        flowNode(
          { id: member.id, label: member.name },
          amount,
          props.factor,
          color(index === 0 ? 'orange' : 'gold'),
        ),
      );
      graph.links.push({ source: member.id, target: HUB, value: amount });
    });
  if (props.summary.monthlyRemainingCents >= 0) return;
  const shortfall = -props.summary.monthlyRemainingCents;
  graph.nodes.push(
    flowNode(
      { id: 'plan-shortfall', label: 'Plan shortfall' },
      shortfall,
      props.factor,
      color('danger'),
    ),
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
  color: ColorReader,
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
      flowNode(
        { id, label },
        amount,
        props.factor,
        color(shades[index % shades.length] ?? 'taupe'),
      ),
      amount,
    );
  });
}

/**
 * Configures readable geometry without changing the graph's financial values.
 * @param graph Mutable chart graph.
 * @param color Chart color token reader.
 * @returns The computed result.
 */
function flowSeries(graph: FlowGraph, color: ColorReader): SankeySeriesOption {
  return {
    id: FLOW_SERIES_ID,
    type: 'sankey',
    data: graph.nodes,
    links: graph.links,
    left: 4,
    top: FLOW_TOP_PADDING,
    bottom: FLOW_BOTTOM_PADDING,
    right: 115,
    nodeWidth: 10,
    nodeGap: FLOW_ROW_GAP,
    draggable: false,
    layoutIterations: 32,
    lineStyle: { color: 'gradient', opacity: 0.32, curveness: 0.5 },
    label: {
      fontSize: 12,
      lineHeight: 17,
      fontWeight: 500,
      color: color('walnut'),
      distance: 10,
      width: 105,
      overflow: 'truncate',
    },
    itemStyle: { borderWidth: 0 },
  };
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
  color: ColorReader,
  height: number,
): EChartsOption {
  const graph: FlowGraph = { nodes: [], links: [] };
  addSources(props, graph, color);
  graph.nodes.push(incomeHub(props, graph, color));
  addExpenses(props, graph, color, categoryCapacity(props, height));
  addAllocation(
    graph,
    flowNode(
      { id: 'plan-goals', label: 'Goals' },
      props.summary.monthlyGoalsCents,
      props.factor,
      color('sage'),
    ),
    props.summary.monthlyGoalsCents,
  );
  addAllocation(
    graph,
    flowNode(
      { id: 'plan-remaining', label: 'Left to plan' },
      props.summary.monthlyRemainingCents,
      props.factor,
      color('walnut-soft'),
    ),
    props.summary.monthlyRemainingCents,
  );
  return {
    animation: false,
    textStyle: { fontFamily: 'DM Sans, sans-serif', color: color('walnut') },
    tooltip: moneyFlowTooltip(graph, props.factor),
    series: [flowSeries(graph, color)],
  };
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
  const minimumNodeSpace = 20;
  const plotHeight = height - FLOW_TOP_PADDING - FLOW_BOTTOM_PADDING - minimumNodeSpace;
  const terminalRows = 1 + Math.floor(Math.max(0, plotHeight) / FLOW_ROW_GAP);
  return Math.max(1, terminalRows - goalCount - remainingCount);
}
