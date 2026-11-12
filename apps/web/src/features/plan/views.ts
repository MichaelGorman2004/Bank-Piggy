/** Names each focused workspace view and its primary action. */
export const VIEW_DETAILS = {
  overview: { label: 'Overview', title: 'Your household plan.', actionLabel: 'Edit plan' },
  flow: { label: 'Money flow', title: 'Money flow', actionLabel: 'Adjust plan' },
  todos: { label: 'To-do', title: 'Your pay-period To-do.', actionLabel: 'Edit pay plan' },
  expenses: { label: 'Expenses', title: 'Your expenses.', actionLabel: 'Add expense' },
  goals: { label: 'Goals', title: 'Your savings & goals.', actionLabel: 'Add goal' },
  targets: {
    label: 'Trips & purchases',
    title: 'Your trips & purchases.',
    actionLabel: 'Add plan',
  },
};

export type View = keyof typeof VIEW_DETAILS;
export type GoalView = Extract<View, 'goals' | 'targets'>;
