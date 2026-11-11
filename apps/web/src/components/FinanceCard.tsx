import { useId, type ReactNode } from 'react';
import styles from './FinanceCard.module.css';

interface FinanceCardProps {
  readonly title: string;
  readonly description: string;
  readonly actions: ReactNode;
  readonly controls?: ReactNode;
  readonly footer?: ReactNode;
  readonly children: ReactNode;
}

/**
 * Keeps card actions visible while its named, keyboard-accessible body scrolls.
 * The parent must allocate a bounded height and allow this card to shrink.
 * @param root0 Card content and fixed controls.
 * @param root0.title Visible title and accessible name for the scrollable body.
 * @param root0.description Short explanation below the title.
 * @param root0.actions Header actions that remain visible.
 * @param root0.controls Optional filters above the scrollable body.
 * @param root0.footer Optional status below the scrollable body.
 * @param root0.children Content that scrolls within the allocated height.
 * @returns A fixed-height card with one scrollable body.
 */
export function FinanceCard({
  title,
  description,
  actions,
  controls,
  footer,
  children,
}: FinanceCardProps) {
  const titleId = useId();
  return (
    <section className={styles.card}>
      <div className={styles.header}>
        <div className={styles.heading}>
          <h2 id={titleId}>{title}</h2>
          <p>{description}</p>
        </div>
        <div className={styles.actions}>{actions}</div>
      </div>
      {controls && <div className={styles.controls}>{controls}</div>}
      <div className={styles.body} role="region" aria-labelledby={titleId} tabIndex={0}>
        {children}
      </div>
      {footer && <div className={styles.footer}>{footer}</div>}
    </section>
  );
}
