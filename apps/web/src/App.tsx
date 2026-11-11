import { Welcome } from './components/Welcome';
import { Workspace } from './Workspace';
import { useBankPiggy } from './useBankPiggy';

/**
 * Opens the household only after both cookie authentication and plan loading succeed.
 * @returns Rendered interface.
 */
export function App() {
  const state = useBankPiggy();
  if (!state.loaded?.session.authenticated || !state.loaded.data)
    return (
      <Welcome
        session={state.loaded?.session ?? null}
        loading={state.loading}
        error={state.error}
        onDemo={() => {
          void state.demo();
        }}
        onRetry={() => {
          void state.reload();
        }}
      />
    );
  return (
    <Workspace
      session={state.loaded.session}
      data={state.loaded.data}
      saving={state.saving}
      loading={state.loading}
      onMonth={state.changeMonth}
      saveError={state.saveError}
      conflict={state.conflict}
      error={state.error}
      onSave={state.save}
      onClearError={state.clearSaveError}
      onReload={() => {
        void state.reload();
      }}
      onSignout={() => {
        void state.signout();
      }}
    />
  );
}
