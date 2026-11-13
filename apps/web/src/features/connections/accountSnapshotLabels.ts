import type { ConnectedAccount } from '@bank-piggy/contracts';
import { CONNECTED_TYPES } from './mappingDraft.ts';

const PROVIDER_ACCOUNT_LABELS: Readonly<Record<string, string>> = {
  checking: 'Checking',
  savings: 'Savings',
  'credit card': 'Credit card',
  'line of credit': 'Line of credit',
  brokerage: 'Brokerage',
  'cash management': 'Cash management',
  'money market': 'Money market',
  cd: 'Certificate of deposit',
  '401k': '401(k)',
  'roth 401k': '401(k)',
  'traditional 401k': '401(k)',
  '403b': '403(b)',
  '457b': '457(b)',
  ira: 'IRA',
  roth: 'IRA',
  'roth ira': 'IRA',
  'traditional ira': 'IRA',
  rollover: 'IRA',
  hsa: 'Health savings account (HSA)',
  mortgage: 'Mortgage',
  auto: 'Auto loan',
  student: 'Student loan',
  depository: 'Bank account',
  investment: 'Investment account',
  credit: 'Credit account',
  loan: 'Loan',
};

/**
 * Uses saved classifications first, then the provider subtype without inspecting account names.
 * Retirement subtypes stay neutral until the member confirms Traditional or Roth treatment.
 * @param account Raw provider account and explicit saved classification.
 * @returns A readable account type with confirmed mapping status when available.
 */
export function accountSnapshotType(account: ConnectedAccount): string {
  const confirmed = CONNECTED_TYPES.find(
    (item) => item.value === account.mapping.classification && item.value !== 'unclassified',
  );
  if (confirmed) return `${confirmed.label} · Confirmed`;
  const raw = account.providerSubtype?.trim() || account.providerType;
  const key = raw.toLowerCase().replaceAll('_', ' ');
  return PROVIDER_ACCOUNT_LABELS[key] ?? key.charAt(0).toUpperCase() + key.slice(1);
}

/**
 * Separates money owed from deposit and investment balances.
 * @param account Raw provider account type.
 * @returns Current and available value labels with credit-specific meaning.
 */
export function accountSnapshotBalanceLabels(account: ConnectedAccount) {
  return {
    current:
      account.providerType === 'credit' || account.providerType === 'loan'
        ? 'Amount owed'
        : 'Current',
    available: account.providerType === 'credit' ? 'Available credit' : 'Available',
  };
}
