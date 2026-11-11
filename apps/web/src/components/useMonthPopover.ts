import { useEffect, useLayoutEffect, useRef, useState } from 'react';

const VIEWPORT_GAP = 12;
const TRIGGER_GAP = 8;

/**
 * Places a top-layer calendar beside its trigger without leaving the visible viewport.
 * @param trigger Calendar opening control.
 * @param popover Open native popover.
 */
function positionCalendar(trigger: HTMLButtonElement, popover: HTMLDivElement) {
  const bounds = trigger.getBoundingClientRect();
  const viewport = window.visualViewport;
  const width = (viewport?.width ?? window.innerWidth) - VIEWPORT_GAP * 2;
  const height = (viewport?.height ?? window.innerHeight) - VIEWPORT_GAP * 2;
  popover.style.maxWidth = `${String(width)}px`;
  popover.style.maxHeight = `${String(height)}px`;
  const calendar = popover.getBoundingClientRect();
  const leftEdge = (viewport?.offsetLeft ?? 0) + VIEWPORT_GAP;
  const topEdge = (viewport?.offsetTop ?? 0) + VIEWPORT_GAP;
  const rightEdge = leftEdge + width;
  const bottomEdge = topEdge + height;
  const below = bounds.bottom + TRIGGER_GAP;
  const above = bounds.top - calendar.height - TRIGGER_GAP;
  const top = below + calendar.height <= bottomEdge ? below : above;
  popover.style.left = `${String(Math.max(leftEdge, Math.min(bounds.left, rightEdge - calendar.width)))}px`;
  popover.style.top = `${String(Math.max(topEdge, Math.min(top, bottomEdge - calendar.height)))}px`;
}

/**
 * Keeps a native month popover within the viewport and focuses its selected month.
 * @returns Trigger and popover references, native visibility state, and close action.
 */
export function useMonthPopover() {
  const trigger = useRef<HTMLButtonElement>(null);
  const popover = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  useLayoutEffect(() => {
    if (!open || !trigger.current || !popover.current) return;
    positionCalendar(trigger.current, popover.current);
    const selected = popover.current.querySelector<HTMLButtonElement>(
      '[aria-pressed="true"]:not(:disabled)',
    );
    const first = popover.current.querySelector<HTMLButtonElement>('[data-month]:not(:disabled)');
    (selected ?? first)?.focus({ preventScroll: true });
  }, [open]);
  useEffect(() => {
    if (!open) return;
    /** Repositions after a viewport change or scrolling inside a dialog. */
    function reposition() {
      if (trigger.current && popover.current) positionCalendar(trigger.current, popover.current);
    }
    window.addEventListener('resize', reposition);
    window.addEventListener('scroll', reposition, true);
    window.visualViewport?.addEventListener('resize', reposition);
    window.visualViewport?.addEventListener('scroll', reposition);
    return () => {
      window.removeEventListener('resize', reposition);
      window.removeEventListener('scroll', reposition, true);
      window.visualViewport?.removeEventListener('resize', reposition);
      window.visualViewport?.removeEventListener('scroll', reposition);
    };
  }, [open]);
  /** Closes after a selection and restores the connected opening control. */
  function close() {
    popover.current?.hidePopover();
    trigger.current?.focus({ preventScroll: true });
  }
  return { trigger, popover, open, setOpen, close };
}
