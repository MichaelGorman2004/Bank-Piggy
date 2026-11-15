import type { EChartsOption } from 'echarts';
import type { SankeySeriesOption } from 'echarts/charts';
import { money } from '../features/plan/format.ts';
import { moneyFlowTooltip } from './moneyFlowTooltip.ts';

export type FlowColorReader = (name: string) => string;
export interface FlowNode {
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

const FLOW_TOP_PADDING = 80;
const FLOW_BOTTOM_PADDING = 22;
const FLOW_ROW_GAP = 36;

/**
 * Keeps editable labels separate from internal identifiers and exact chart cents.
 * @param node Stable identifier and display label.
 * @param node.id Internal chart identifier.
 * @param node.label Display label.
 * @param amount Canonical cents before any planned period scale.
 * @param presentation Label scale and themed node color.
 * @param presentation.factor Selected label scale.
 * @param presentation.color Node color from the theme.
 * @returns A labeled Sankey node.
 */
export function flowNode(
  node: { readonly id: string; readonly label: string },
  amount: number,
  presentation: { readonly factor: number; readonly color: string },
): FlowNode {
  return {
    name: node.id,
    itemStyle: { color: presentation.color },
    label: { formatter: `${node.label}\n${money(amount, presentation.factor)}` },
  };
}

/**
 * Places the complete hub label above the bar so thin streams cannot hide it.
 * @param node Stable identifier and display label.
 * @param node.id Internal chart identifier.
 * @param node.label Display label.
 * @param amount Canonical cents before any planned period scale.
 * @param presentation Label scale and themed node color.
 * @param presentation.factor Selected label scale.
 * @param presentation.color Node color from the theme.
 * @returns A central Sankey node with a full label above its geometry.
 */
export function flowHub(
  node: { readonly id: string; readonly label: string },
  amount: number,
  presentation: { readonly factor: number; readonly color: string },
): FlowNode {
  const hub = flowNode(node, amount, presentation);
  return {
    ...hub,
    label: {
      ...hub.label,
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
 * Fits complete labels above and beside the shared Sankey geometry.
 * @param graph Canonical graph nodes and links.
 * @param color Chart color token reader.
 * @param id Stable chart series identifier.
 * @returns Shared series style with unchanged financial values.
 */
function flowSeries(graph: FlowGraph, color: FlowColorReader, id: string): SankeySeriesOption {
  return {
    id,
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
 * Applies the same theme, tooltip safety, and geometry to planned and recorded flows.
 * @param graph Exact nodes and links supplied by the owning flow.
 * @param color Chart color token reader.
 * @param factor Selected label scale.
 * @param id Stable series identifier.
 * @returns A complete Sankey option.
 */
export function moneyFlowChartOption(
  graph: FlowGraph,
  color: FlowColorReader,
  factor: number,
  id: string,
): EChartsOption {
  return {
    animation: false,
    textStyle: { fontFamily: 'DM Sans, sans-serif', color: color('walnut') },
    tooltip: moneyFlowTooltip(graph, factor),
    series: [flowSeries(graph, color, id)],
  };
}

/**
 * Reserves fixed terminal rows before compacting the remaining categories.
 * @param height Measured chart height in CSS pixels.
 * @param reservedRows Visible rows that must not be combined.
 * @returns Category capacity, retaining at least one row.
 */
export function flowCategoryCapacity(height: number, reservedRows: number): number {
  const minimumNodeSpace = 20;
  const plotHeight = height - FLOW_TOP_PADDING - FLOW_BOTTOM_PADDING - minimumNodeSpace;
  const terminalRows = 1 + Math.floor(Math.max(0, plotHeight) / FLOW_ROW_GAP);
  return Math.max(1, terminalRows - reservedRows);
}
