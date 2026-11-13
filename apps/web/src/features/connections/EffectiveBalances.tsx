import type {
  BalanceTotals,
  EffectiveAccount,
  EffectiveBalances as BalanceProjection,
  Household,
} from '@bank-piggy/contracts';
import { PiggyBank, TrendingUp, Wallet, HeartPulse } from 'lucide-react';
import { HOUSEHOLD_SCOPE, money } from '../plan/format';
import { BalanceAvailability } from './BalanceAvailability';
import { connectedMoney } from './presentation';
import { CONNECTED_TYPES } from './mappingDraft';
import balanceStyles from '../../components/BalancePanel.module.css';
import rowStyles from '../../components/DashboardRows.module.css';
import styles from './Connections.module.css';

const TOTAL_ROWS = [
  { field: 'checkingCents', label: 'Checking', Icon: Wallet },
  { field: 'savingsCents', label: 'Savings', Icon: PiggyBank },
  { field: 'investmentsCents', label: 'Investments', Icon: TrendingUp },
  { field: 'hsaCents', label: 'Health savings', Icon: HeartPulse },
] satisfies ReadonlyArray<{
  readonly field: keyof BalanceTotals;
  readonly label: string;
  readonly Icon: typeof Wallet;
}>;

/**
 * Shows the server's reconciled USD totals and labels underlying account snapshots separately.
 * @param props Authorized projection, saved members, and selected presentation scope.
 * @param props.balances Server-owned totals and account inclusion decisions.
 * @param props.household Saved member names.
 * @param props.scope Household or personal presentation scope.
 * @returns Balance totals with explicit source and full joint-snapshot labels.
 */
export function EffectiveBalances(props: {
  readonly balances: BalanceProjection;
  readonly household: Household;
  readonly scope: string;
}) {
  const totals =
    props.scope === HOUSEHOLD_SCOPE
      ? props.balances.household
      : props.balances.members[props.scope];
  const accounts = props.balances.accounts.filter(
    (account) =>
      account.included &&
      (props.scope === HOUSEHOLD_SCOPE ||
        account.ownership === props.scope ||
        account.ownership === 'shared'),
  );
  return (
    <>
      <BalanceAvailability scope={props.scope} />
      {TOTAL_ROWS.map(({ field, label, Icon }) => (
        <div className={rowStyles.balanceRow} key={field}>
          <span className={rowStyles.balanceLabel}>
            <Icon size={18} />
            {label}
          </span>
          <strong>{money(totals[field])}</strong>
        </div>
      ))}
      {!accounts.length ? (
        <p className={balanceStyles.empty}>No included account balances.</p>
      ) : (
        <>
          <p className={styles.meta}>
            Account snapshots · Joint rows show the full account balance.
          </p>
          <dl className={balanceStyles.breakdown}>
            {accounts.map((account) => (
              <AccountSnapshot key={account.id} account={account} household={props.household} />
            ))}
          </dl>
        </>
      )}
    </>
  );
}

/**
 * Labels source and ownership without deriving personal shares in the browser.
 * @param props One included account and saved member names.
 * @param props.account Full account snapshot selected by the server.
 * @param props.household Saved household members.
 * @returns One source-labeled snapshot row.
 */
function AccountSnapshot(props: {
  readonly account: EffectiveAccount;
  readonly household: Household;
}) {
  const account = props.account;
  const owner =
    account.ownership === 'shared'
      ? 'Joint / shared'
      : props.household.members.find((member) => member.id === account.ownership)?.name;
  const type = CONNECTED_TYPES.find((type) => type.value === account.classification)?.label;
  return (
    <div>
      <dt>
        {account.name}
        <span>
          {type} · {owner} · {account.source === 'plaid' ? 'Connected' : 'Manual'}
        </span>
      </dt>
      <dd>{connectedMoney(account.balanceCents, account.currency)}</dd>
    </div>
  );
}
