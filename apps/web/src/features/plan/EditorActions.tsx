import { Trash2 } from 'lucide-react';
import styles from './Editors.module.css';

interface EditorActionsProps {
  readonly kind: 'expense' | 'goal';
  readonly busy: boolean;
  readonly onClose: () => void;
  readonly onDelete?: () => void;
}

/**
 * Keeps save, cancel, and removal actions consistent across single-item editors.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function EditorActions(props: EditorActionsProps) {
  return (
    <div className={styles.actions}>
      {props.onDelete && (
        <button
          type="button"
          className={styles.danger}
          disabled={props.busy}
          onClick={props.onDelete}
        >
          <Trash2 size={15} />
          Remove
        </button>
      )}
      <button
        type="button"
        className={styles.secondary}
        disabled={props.busy}
        onClick={props.onClose}
      >
        Cancel
      </button>
      <button className={styles.button} disabled={props.busy}>
        {props.busy ? 'Saving…' : `Save ${props.kind}`}
      </button>
    </div>
  );
}
