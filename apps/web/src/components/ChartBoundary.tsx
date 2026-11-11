import { Component, type ReactNode } from 'react';
import styles from './Dashboard.module.css';

interface ChartBoundaryProps {
  readonly children: ReactNode;
}
interface ChartBoundaryState {
  readonly failed: boolean;
}

/** Contains a chart-loading failure so the editable plan remains usable. */
export class ChartBoundary extends Component<ChartBoundaryProps, ChartBoundaryState> {
  state: ChartBoundaryState = { failed: false };

  /**
   * Replaces a failed lazy chart with an actionable message.
   * @returns Chart failure state.
   */
  static getDerivedStateFromError(): ChartBoundaryState {
    return { failed: true };
  }

  /**
   * Keeps chart failures separate from the rest of the household dashboard.
   * @returns The chart or its reload action.
   */
  render(): ReactNode {
    if (!this.state.failed) return this.props.children;
    return (
      <div className={styles.empty} role="alert">
        <p>The money flow could not load. You can still use the other views.</p>
        <button
          className={styles.textButton}
          onClick={() => {
            window.location.reload();
          }}
        >
          Reload page
        </button>
      </div>
    );
  }
}
