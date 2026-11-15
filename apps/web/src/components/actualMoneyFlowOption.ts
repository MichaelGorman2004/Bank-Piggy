import type { ActivityBankFlow, ActivityFlowAmount } from '@bank-piggy/contracts';
import type { EChartsOption } from 'echarts';
import { compactFlowCategories, type FlowCategory } from './moneyFlowCategories.ts';
import {
  flowCategoryCapacity,
  flowHub,
  flowNode,
  moneyFlowChartOption,
  type FlowColorReader,
  type FlowGraph,
} from './moneyFlowChart.ts';

const HUB = 'actual-bank-hub';
const FLOW_SERIES_ID = 'actual-bank-money-flow';
const SPENDING_PREFIX = 'spending-';
const REFUND_PREFIX = 'refund-';
const NODE_SHADES = ['orange', 'gold', 'taupe', 'pink'];

/**
 * Combines small categories while preserving explicit income, reversals, gap, and surplus.
 * @param rows Exact server flow rows in display order.
 * @param prefix Server category identifier prefix.
 * @param height Measured chart height in CSS pixels.
 * @returns Readable rows with all canonical cents retained.
 */
function compactRows(
  rows: ReadonlyArray<ActivityFlowAmount>,
  prefix: string,
  height: number,
): FlowCategory[] {
  const categories = rows.filter((row) => row.id.startsWith(prefix));
  const fixedCount = rows.length - categories.length;
  const compacted = compactFlowCategories(
    categories.map((row) => ({ id: row.id, label: row.name, amount: row.amountCents })),
    flowCategoryCapacity(height, fixedCount),
    prefix === REFUND_PREFIX ? 'Other refunds' : 'Other costs',
  );
  let inserted = false;
  return rows.flatMap((row) => {
    if (!row.id.startsWith(prefix))
      return [{ id: row.id, label: row.name, amount: row.amountCents }];
    if (inserted) return [];
    inserted = true;
    return compacted;
  });
}

/**
 * Keeps server balancing rows visually distinct from recorded income and categories.
 * @param id Canonical row identifier.
 * @param index Display row index.
 * @returns Theme token name.
 */
function rowShade(id: string, index: number): string {
  if (id === 'inflow-gap' || id === 'income-reversals') return 'danger';
  if (id === 'surplus') return 'sage';
  return NODE_SHADES[index % NODE_SHADES.length] ?? 'taupe';
}

/**
 * Creates one side of the graph without classifying or recalculating recorded activity.
 * @param graph Mutable chart graph.
 * @param rows Canonical flow rows after presentation-only compaction.
 * @param direction Flow side relative to the central bar.
 * @param color Chart color token reader.
 */
function addRows(
  graph: FlowGraph,
  rows: ReadonlyArray<FlowCategory>,
  direction: 'inflow' | 'outflow',
  color: FlowColorReader,
) {
  rows.forEach((row, index) => {
    const id = `actual-${direction}-${row.id}`;
    graph.nodes.push(
      flowNode({ id, label: row.label }, row.amount, {
        factor: 1,
        color: color(rowShade(row.id, index)),
      }),
    );
    graph.links.push({
      source: direction === 'inflow' ? id : HUB,
      target: direction === 'inflow' ? HUB : id,
      value: row.amount,
    });
  });
}

/**
 * Draws only the balanced, server-calculated posted bank flow for the selected month.
 * @param data Canonical bank flow; investment contributions never enter this graph.
 * @param color Chart color token reader.
 * @param height Measured chart height in CSS pixels.
 * @returns A themed Sankey option with canonical cent-valued edges.
 */
export function actualMoneyFlowOption(
  data: ActivityBankFlow,
  color: FlowColorReader,
  height: number,
): EChartsOption {
  const graph: FlowGraph = { nodes: [], links: [] };
  if (data.totalCents === 0) return moneyFlowChartOption(graph, color, 1, FLOW_SERIES_ID);
  addRows(graph, compactRows(data.inflows, REFUND_PREFIX, height), 'inflow', color);
  graph.nodes.push(
    flowHub({ id: HUB, label: 'Recorded bank flow' }, data.totalCents, {
      factor: 1,
      color: color('taupe'),
    }),
  );
  addRows(graph, compactRows(data.outflows, SPENDING_PREFIX, height), 'outflow', color);
  return moneyFlowChartOption(graph, color, 1, FLOW_SERIES_ID);
}
