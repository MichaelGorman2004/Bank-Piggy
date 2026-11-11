import type { TooltipComponentFormatterCallbackParams, TooltipComponentOption } from 'echarts';
import { format } from 'echarts/core';
import { money } from '../features/plan/format.ts';
import type { FlowGraph } from './moneyFlowOption';

const TOOLTIP_TEXT_WIDTH = 200;
const GRAPHEMES = new Intl.Segmenter(undefined, { granularity: 'grapheme' });

/**
 * Wraps measured text while preserving complete Unicode characters and the full label.
 * @param text Visible source, destination, or node label.
 * @returns Plain text sized for the narrowest supported chart.
 */
function wrapLabel(text: string): string {
  const lines: string[] = [];
  let line = '';
  for (const { segment } of GRAPHEMES.segment(text)) {
    if (segment === '\n') {
      lines.push(line);
      line = '';
      continue;
    }
    // ECharts measures its rich-text tooltip with this same default font.
    if (line && format.getTextRect(line + segment).width > TOOLTIP_TEXT_WIDTH) {
      lines.push(line);
      line = segment;
    } else line += segment;
  }
  lines.push(line);
  return lines.join('\n');
}

/**
 * Reads a visible label without exposing internal identifiers.
 * @param graph Chart nodes and links.
 * @param id Internal graph identifier.
 * @returns Plain display text.
 */
function displayLabel(graph: FlowGraph, id: string): string {
  const formatted = graph.nodes.find((node) => node.name === id)?.label.formatter;
  const label = formatted?.slice(0, formatted.lastIndexOf('\n'));
  return label || 'Your plan';
}

/**
 * Formats a node or edge using plain text in ECharts' non-HTML renderer.
 * @param params ECharts tooltip item.
 * @param graph Chart nodes and links.
 * @param factor Selected plan-rate scale.
 * @returns Human-readable tooltip text.
 */
function tooltipText(
  params: TooltipComponentFormatterCallbackParams,
  graph: FlowGraph,
  factor: number,
): string {
  const item = Array.isArray(params) ? params.at(0) : params;
  if (!item) return '';
  if (item.dataType === 'edge') {
    const link = graph.links.at(item.dataIndex);
    if (!link) return '';
    const label = `${displayLabel(graph, link.source)} → ${displayLabel(graph, link.target)}`;
    return `${wrapLabel(label)}\n${money(link.value, factor, true)}`;
  }
  const amount = typeof item.value === 'number' ? money(item.value, factor, true) : '';
  return `${wrapLabel(displayLabel(graph, item.name))}\n${amount}`;
}

/**
 * Uses rich-text rendering so editable names cannot inject HTML into a tooltip.
 * @param graph Canonical chart nodes and links.
 * @param factor Selected plan-rate scale.
 * @returns Tooltip configuration.
 */
export function moneyFlowTooltip(graph: FlowGraph, factor: number): TooltipComponentOption {
  return {
    trigger: 'item',
    confine: true,
    renderMode: 'richText',
    formatter: (params) => tooltipText(params, graph, factor),
  };
}
