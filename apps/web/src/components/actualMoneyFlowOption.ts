import type { ActivityBankFlow, ActivityFlowAmount } from '@bank-piggy/contracts';
import type { EChartsOption } from 'echarts';
import {
  flowHub,
  flowNode,
  moneyFlowChartOption,
  type FlowColorReader,
  type FlowGraph,
} from './moneyFlowChart.ts';

const HUB = 'actual-bank-hub';
const FLOW_SERIES_ID = 'actual-bank-money-flow';

/** Keeps aggregate differences visually distinct from recorded income and spending.
 * @param id Canonical aggregate flow identifier.
 * @returns Theme token name.
 */
function rowShade(id: string): string {
  if (id === 'inflow-gap' || id === 'income-reversals') return 'danger';
  if (id === 'surplus') return 'sage';
  if (id === 'income') return 'gold';
  if (id === 'net-refunds') return 'pink';
  return 'orange';
}

/** Creates one graph side directly from canonical aggregate amounts.
 * @param graph Mutable chart graph.
 * @param rows Server-calculated aggregate rows.
 * @param direction Flow side relative to the central bar.
 * @param color Chart color token reader.
 */
function addRows(
  graph: FlowGraph,
  rows: ReadonlyArray<ActivityFlowAmount>,
  direction: 'inflow' | 'outflow',
  color: FlowColorReader,
) {
  rows.forEach((row) => {
    const id = `actual-${direction}-${row.id}`;
    graph.nodes.push(
      flowNode({ id, label: row.name }, row.amountCents, {
        factor: 1,
        color: color(rowShade(row.id)),
      }),
    );
    graph.links.push({
      source: direction === 'inflow' ? id : HUB,
      target: direction === 'inflow' ? HUB : id,
      value: row.amountCents,
    });
  });
}

/** Draws aggregate posted bank flow without purchase or category detail.
 * @param data Canonical bank totals; investment contributions stay separate.
 * @param color Chart color token reader.
 * @returns A themed Sankey option with canonical cent-valued edges.
 */
export function actualMoneyFlowOption(
  data: ActivityBankFlow,
  color: FlowColorReader,
): EChartsOption {
  const graph: FlowGraph = { nodes: [], links: [] };
  if (data.totalCents === 0) return moneyFlowChartOption(graph, color, 1, FLOW_SERIES_ID);
  addRows(graph, data.inflows, 'inflow', color);
  graph.nodes.push(
    flowHub({ id: HUB, label: 'Recorded bank flow' }, data.totalCents, {
      factor: 1,
      color: color('taupe'),
    }),
  );
  addRows(graph, data.outflows, 'outflow', color);
  return moneyFlowChartOption(graph, color, 1, FLOW_SERIES_ID);
}
