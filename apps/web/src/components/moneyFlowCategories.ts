export interface FlowCategory {
  readonly id: string;
  readonly label: string;
  readonly amount: number;
}

/**
 * Fits category labels into the available rows without dropping any allocated cents.
 * @param categories Exact expense totals in their original display order.
 * @param capacity Maximum visible category rows; at least one row is retained.
 * @returns Original rows when they fit, or largest categories and an explicit remainder.
 */
export function compactFlowCategories(
  categories: ReadonlyArray<FlowCategory>,
  capacity: number,
): FlowCategory[] {
  const limit = Math.max(1, Math.floor(capacity));
  if (categories.length <= limit) return [...categories];
  const ranked = [...categories].sort((first, second) => second.amount - first.amount);
  const visible = ranked.slice(0, limit - 1);
  const remainder = ranked.slice(limit - 1);
  return [
    ...visible,
    {
      id: 'grouped-expenses',
      label: `Other costs (${String(remainder.length)})`,
      amount: remainder.reduce((sum, category) => sum + category.amount, 0),
    },
  ];
}
