import { createContext, useContext, type ReactNode } from 'react';
import { useConnections } from './useConnections';

type ConnectionData = ReturnType<typeof useConnections> & { readonly csrfToken: string };
const Context = createContext<ConnectionData | null>(null);

/**
 * Shares one authoritative balance projection between Overview tables and account management.
 * @param props Current plan revision and financial workspace.
 * @param props.revision Saved household revision.
 * @param props.csrfToken Current session mutation token.
 * @param props.children Financial views that consume account balances.
 * @returns Views with one shared connection request owner.
 */
export function ConnectionsProvider(props: {
  readonly revision: number;
  readonly csrfToken: string;
  readonly children: ReactNode;
}) {
  const state = useConnections(props.revision);
  return <Context value={{ ...state, csrfToken: props.csrfToken }}>{props.children}</Context>;
}

/**
 * Reads the shared authorized connection projection without recalculating totals.
 * @returns Current connection snapshot and request operations.
 */
export function useConnectionData(): ConnectionData {
  const state = useContext(Context);
  if (!state) throw new Error('Connected account views require their household provider.');
  return state;
}
