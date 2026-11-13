import { useEffect } from 'react';

const POINTER_FOCUS_ATTRIBUTE = 'data-pointer-focus';
const MODIFIER_KEYS = new Set(['Alt', 'Control', 'Meta', 'Shift']);

/** Keeps pointer focus quiet and native keyboard focus visible across every app route. */
export function useFocusVisibility(): void {
  useEffect(() => startFocusVisibility(), []);
}

/**
 * Tracks input before controls or dialogs move focus, without changing native actions.
 * The unmarked document retains native focus-visible styling as the default.
 * @param page Document that owns the application's controls and dialogs.
 * @returns Cleanup that removes listeners and restores the default focus style.
 */
export function startFocusVisibility(page: Document = document): () => void {
  const root = page.documentElement;
  const captureOptions = { capture: true };
  /** Removes pointer focus decoration, including Safari's native select focus ring. */
  const pointer = () => {
    root.setAttribute(POINTER_FOCUS_ATTRIBUTE, '');
  };
  /**
   * Restores keyboard styling before navigation or activation can move focus.
   * @param event Keyboard input; modifier presses alone do not move focus.
   */
  const keyboard = (event: KeyboardEvent) => {
    if (MODIFIER_KEYS.has(event.key)) return;
    root.removeAttribute(POINTER_FOCUS_ATTRIBUTE);
  };
  page.addEventListener('pointerdown', pointer, captureOptions);
  page.addEventListener('keydown', keyboard, captureOptions);
  return () => {
    page.removeEventListener('pointerdown', pointer, captureOptions);
    page.removeEventListener('keydown', keyboard, captureOptions);
    root.removeAttribute(POINTER_FOCUS_ATTRIBUTE);
  };
}
