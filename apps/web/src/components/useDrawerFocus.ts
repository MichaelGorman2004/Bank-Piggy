import { useEffect, type RefObject } from 'react';

/**
 * Keeps keyboard focus inside an open mobile drawer and restores its trigger on close.
 * @param ref Drawer element reference.
 * @param open Whether the mobile drawer is open.
 * @param onClose Closes the current dialog or drawer.
 */
export function useDrawerFocus(
  ref: RefObject<HTMLElement | null>,
  open: boolean,
  onClose: () => void,
) {
  useEffect(() => {
    if (!open) return;
    const previous = document.activeElement;
    const controls = ref.current?.querySelectorAll<HTMLElement>('button:not([disabled]), a[href]');
    const first = controls?.[0];
    const last = controls?.[controls.length - 1];
    first?.focus();
    /**
     * Contains keyboard focus and handles the drawer’s Escape action.
     * @param event Browser form or keyboard event.
     */
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        event.preventDefault();
        onClose();
        return;
      }
      if (event.key !== 'Tab') return;
      const target = event.shiftKey ? first : last;
      if (document.activeElement !== target) return;
      event.preventDefault();
      if (event.shiftKey) last?.focus();
      else first?.focus();
    }
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('keydown', onKey);
      if (previous instanceof HTMLElement) previous.focus();
    };
  }, [open, ref, onClose]);
}
