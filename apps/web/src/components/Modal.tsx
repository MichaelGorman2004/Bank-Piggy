import { useEffect, useRef, type ReactNode } from 'react';
import { X } from 'lucide-react';
import styles from './Modal.module.css';

interface ModalProps {
  readonly title: string;
  readonly description?: string;
  readonly children: ReactNode;
  readonly onClose: () => void;
  readonly busy?: boolean;
  readonly wide?: boolean;
  readonly suspended?: boolean;
}

/**
 * Opens the native dialog and restores its connected opening control on unmount.
 * @param suspended Whether an external consent dialog temporarily owns focus.
 * @returns Dialog element reference.
 */
function useModalFocus(suspended: boolean) {
  const ref = useRef<HTMLDialogElement>(null);
  const pausedFocus = useRef<HTMLElement | null>(null);
  useEffect(() => {
    const dialog = ref.current;
    const previous = document.activeElement;
    return () => {
      dialog?.close();
      if (previous instanceof HTMLElement && previous.isConnected)
        previous.focus({ preventScroll: true });
    };
  }, []);
  useEffect(() => {
    const dialog = ref.current;
    if (suspended) {
      const active = document.activeElement;
      if (active instanceof HTMLElement && dialog?.contains(active)) pausedFocus.current = active;
      dialog?.close();
      return;
    }
    dialog?.showModal();
    focusDialog(dialog, pausedFocus.current);
  }, [suspended]);
  return ref;
}

/**
 * Restores the suspended control or focuses the first editable native field.
 * @param dialog Mounted native dialog.
 * @param previous Control that had focus before external provider consent.
 */
function focusDialog(dialog: HTMLDialogElement | null, previous: HTMLElement | null) {
  const target = previous?.isConnected
    ? previous
    : dialog?.querySelector<HTMLElement>(
        'input:not(:disabled), select:not(:disabled), textarea:not(:disabled)',
      );
  target?.focus({ preventScroll: true });
}

/**
 * Keeps the dialog header fixed and scrolling inside its body; cleanup restores focus.
 * @param root0 Component values and event handlers.
 * @param root0.title Dialog title.
 * @param root0.description Dialog explanation.
 * @param root0.children Dialog body.
 * @param root0.onClose Closes the current dialog or drawer.
 * @param root0.busy Whether persistence is in progress.
 * @param root0.wide Whether the dialog uses the wider layout.
 * @param root0.suspended Whether an external consent dialog temporarily owns focus.
 * @returns Rendered interface.
 */
export function Modal({
  title,
  description,
  children,
  onClose,
  busy = false,
  wide = false,
  suspended = false,
}: ModalProps) {
  const ref = useModalFocus(suspended);
  return (
    <dialog
      ref={ref}
      className={`${styles.dialog} ${wide ? styles.wide : ''}`}
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
      aria-labelledby="modal-title"
      aria-describedby={description ? 'modal-description' : undefined}
    >
      <div className={styles.header}>
        <div>
          <h2 id="modal-title">{title}</h2>
          {description && <p id="modal-description">{description}</p>}
        </div>
        <button
          className={styles.close}
          aria-label="Close dialog"
          disabled={busy}
          onClick={onClose}
        >
          <X size={20} />
        </button>
      </div>
      <div className={styles.body} role="region" aria-label={`${title} content`} tabIndex={0}>
        {children}
      </div>
    </dialog>
  );
}
