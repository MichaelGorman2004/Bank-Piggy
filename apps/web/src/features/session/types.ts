import type { Dispatch, SetStateAction } from 'react';
import type { HouseholdResponse, Session } from '@bank-piggy/contracts';

export interface LoadedState {
  readonly session: Session;
  readonly data: HouseholdResponse | null;
}

export type LoadedStateSetter = Dispatch<SetStateAction<LoadedState | null>>;
