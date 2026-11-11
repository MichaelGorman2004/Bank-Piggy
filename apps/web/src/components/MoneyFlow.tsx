import { useEffectEvent, useLayoutEffect, useRef } from 'react';
import { dispose, init, use as registerCharts, type EChartsType } from 'echarts/core';
import { SankeyChart } from 'echarts/charts';
import { TooltipComponent } from 'echarts/components';
import { SVGRenderer } from 'echarts/renderers';
import { money } from '../features/plan/format';
import { moneyFlowOption, type MoneyFlowProps } from './moneyFlowOption';
import styles from './MoneyFlow.module.css';

registerCharts([SankeyChart, TooltipComponent, SVGRenderer]);

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
 * @param changed Whether inputs changed independently of the element's dimensions.
 */
function renderMoneyFlow(renderer: FlowRenderer, props: MoneyFlowProps, changed: boolean) {
  const { chart, element } = renderer;
  const width = element.clientWidth;
  const height = element.clientHeight;
  const resized = width !== renderer.width || height !== renderer.height;
  if (!changed && !resized) return;
  try {
    if (resized) chart.resize({ width, height });
    const computed = getComputedStyle(document.documentElement);
    chart.setOption(
      moneyFlowOption(props, (name) => computed.getPropertyValue(`--${name}`).trim(), height),
    );
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
 * @param visible Whether the populated chart element exists in this render.
 * @returns The chart element reference.
 */
function useMoneyFlow(props: MoneyFlowProps, visible: boolean) {
  const ref = useRef<HTMLDivElement>(null);
  const current = useRef<FlowRenderer | null>(null);
  const { data, summary, scope, factor } = props;
  const update = useEffectEvent((renderer: FlowRenderer, changed: boolean) => {
    if (current.current !== renderer || renderer.chart.isDisposed()) return;
    renderMoneyFlow(renderer, props, changed);
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
  }, [data, summary, scope, factor, visible]);
  return ref;
}

/**
 * Offers the same financial totals as accessible text outside the visual chart.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
function FlowDescription(props: MoneyFlowProps) {
  const { summary, factor } = props;
  return (
    <p className={styles.accessible}>
      Planned income {money(summary.monthlyTakeHomeCents, factor)}. Expenses{' '}
      {money(summary.monthlyExpensesCents, factor)}. Goals{' '}
      {money(summary.monthlyGoalsCents, factor)}.{' '}
      {summary.monthlyRemainingCents < 0 ? 'Shortfall' : 'Left to plan'}{' '}
      {money(Math.abs(summary.monthlyRemainingCents), factor)}. Each stream's width shows its
      amount.
    </p>
  );
}

/**
 * Shows server-calculated income and allocations; shortfalls remain explicit.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function MoneyFlow(props: MoneyFlowProps) {
  const empty =
    props.summary.monthlyTakeHomeCents +
      props.summary.monthlyExpensesCents +
      props.summary.monthlyGoalsCents ===
    0;
  const ref = useMoneyFlow(props, !empty);
  return (
    <div className={styles.wrap}>
      {empty ? (
        <div className={styles.empty}>
          Add your take-home pay and expenses to see your money flow.
        </div>
      ) : (
        <div ref={ref} className={styles.chart} aria-hidden="true" />
      )}
      <FlowDescription {...props} />
    </div>
  );
}
