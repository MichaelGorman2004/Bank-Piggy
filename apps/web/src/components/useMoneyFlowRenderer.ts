import { useEffectEvent, useLayoutEffect, useRef } from 'react';
import { dispose, init, use as registerCharts, type EChartsType } from 'echarts/core';
import { SankeyChart } from 'echarts/charts';
import { TooltipComponent } from 'echarts/components';
import { SVGRenderer } from 'echarts/renderers';
import type { EChartsOption } from 'echarts';
import type { FlowColorReader } from './moneyFlowChart';

registerCharts([SankeyChart, TooltipComponent, SVGRenderer]);

type FlowOption<Input> = (props: Input, color: FlowColorReader, height: number) => EChartsOption;

interface FlowRenderer {
  readonly chart: EChartsType;
  readonly element: HTMLDivElement;
  readonly resize: ResizeObserver;
  width: number;
  height: number;
}

/**
 * Disconnects size notifications and releases the owned renderer.
 * @param renderer Renderer owned by the current chart element.
 */
function disposeRenderer(renderer: FlowRenderer) {
  renderer.resize.disconnect();
  if (!renderer.chart.isDisposed()) renderer.chart.dispose();
}

/**
 * Creates one renderer and observer for an actual visible chart element.
 * @param element Chart element owned by this component.
 * @param onResize Reads current committed inputs when the element changes size.
 * @returns Resources that must be released when this element disappears.
 */
function createRenderer(
  element: HTMLDivElement,
  onResize: (renderer: FlowRenderer) => void,
): FlowRenderer {
  let resize: ResizeObserver | undefined;
  try {
    const chart = init(element, undefined, { renderer: 'svg' });
    const renderer: FlowRenderer = {
      chart,
      element,
      resize: new ResizeObserver(() => {
        onResize(renderer);
      }),
      width: chart.getWidth(),
      height: chart.getHeight(),
    };
    resize = renderer.resize;
    resize.observe(element);
    return renderer;
  } catch (error) {
    try {
      resize?.disconnect();
      dispose(element);
    } catch {
      // Preserve the initialization error if cleanup also fails.
    }
    throw error;
  }
}

/**
 * Updates the existing series; unchanged observer notifications do no work.
 * @param renderer Renderer for the current visible element.
 * @param props Current committed financial inputs and scope.
 * @param option Pure option builder for the owning flow.
 * @param changed Whether inputs changed independently of the element's dimensions.
 */
function renderMoneyFlow<Input>(
  renderer: FlowRenderer,
  props: Input,
  option: FlowOption<Input>,
  changed: boolean,
) {
  const { chart, element } = renderer;
  const width = element.clientWidth;
  const height = element.clientHeight;
  const resized = width !== renderer.width || height !== renderer.height;
  if (!changed && !resized) return;
  try {
    if (resized) chart.resize({ width, height });
    const computed = getComputedStyle(document.documentElement);
    chart.setOption(option(props, (name) => computed.getPropertyValue(`--${name}`).trim(), height));
    renderer.width = width;
    renderer.height = height;
  } catch (error) {
    try {
      disposeRenderer(renderer);
    } catch {
      // Preserve the rendering error if cleanup also fails.
    }
    throw error;
  }
}

/**
 * Keeps the renderer for its element's lifetime and updates it before browser paint.
 * @param props Component values and event handlers.
 * @param option Pure option builder for the owning flow.
 * @param visible Whether the populated chart element exists in this render.
 * @returns The chart element reference.
 */
export function useMoneyFlowRenderer<Input>(
  props: Input,
  option: FlowOption<Input>,
  visible: boolean,
) {
  const ref = useRef<HTMLDivElement>(null);
  const current = useRef<FlowRenderer | null>(null);
  const update = useEffectEvent((renderer: FlowRenderer, changed: boolean) => {
    if (current.current !== renderer || renderer.chart.isDisposed()) return;
    renderMoneyFlow(renderer, props, option, changed);
  });
  useLayoutEffect(() => {
    const element = ref.current;
    if (!visible || !element) return;
    const renderer = createRenderer(element, (instance) => {
      update(instance, false);
    });
    current.current = renderer;
    return () => {
      if (current.current === renderer) current.current = null;
      disposeRenderer(renderer);
    };
  }, [visible]);
  useLayoutEffect(() => {
    if (current.current) update(current.current, true);
  }, [props, option, visible]);
  return ref;
}
