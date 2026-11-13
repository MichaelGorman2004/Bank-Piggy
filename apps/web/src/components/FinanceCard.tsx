import { useId, type ReactNode } from 'react';
import styles from './FinanceCard.module.css';

interface FinanceCardProps {
  readonly title: string;
  readonly description: string;
  readonly actions: ReactNode;
  readonly controls?: ReactNode;
  readonly footer?: ReactNode;
  readonly children: ReactNode;
  readonly className?: string;
  readonly scrollable?: boolean;
}

/**
 * Keeps card actions visible around a named body with optional internal scrolling.
 * The parent must allocate a bounded height and allow this card to shrink.
 * @param root0 Card content and fixed controls.
 * @param root0.title Visible title and accessible name for the scrollable body.
 * @param root0.description Short explanation below the title.
 * @param root0.actions Header actions that remain visible.
 * @param root0.controls Optional filters above the scrollable body.
 * @param root0.footer Optional status below the scrollable body.
 * @param root0.children Content that scrolls within the allocated height.
 * @param root0.className Feature-owned layout styles for the card.
 * @param root0.scrollable Whether the body scrolls; false requires content that fits its height.
 * @returns A fixed-height card with one optionally scrollable body.
 */
export function FinanceCard({
  title,
  description,
  actions,
  controls,
  footer,
  children,
  className = '',
  scrollable = true,
}: FinanceCardProps) {
  const titleId = useId();
  return (
    <section className={`${styles.card} ${className}`}>
      <div className={styles.header}>
        <div className={styles.heading}>
          <h2 id={titleId}>{title}</h2>
          <p data-card-description>{description}</p>
        </div>
        <div className={styles.actions}>{actions}</div>
      </div>
      {controls && <div className={styles.controls}>{controls}</div>}
      <div
        className={`${styles.body} ${scrollable ? '' : styles.staticBody}`}
        role="region"
        aria-labelledby={titleId}
        tabIndex={scrollable ? 0 : undefined}
      >
        {children}
      </div>
      {footer && <div className={styles.footer}>{footer}</div>}
    </section>
  );
}
