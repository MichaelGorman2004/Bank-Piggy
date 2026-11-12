import { useRef } from 'react';
import {
  ChartNoAxesCombined,
  ListChecks,
  House,
  LogOut,
  Plane,
  Sprout,
  Wallet,
  X,
} from 'lucide-react';
import type { Member, Session } from '@bank-piggy/contracts';
import { VIEW_DETAILS, type View } from '../features/plan/views';
import { useDrawerFocus } from './useDrawerFocus';
import styles from './Sidebar.module.css';
import sharingStyles from '../features/sharing/Sharing.module.css';

interface SidebarProps {
  readonly session: Session;
  readonly members: readonly Member[];
  readonly view: View;
  readonly open: boolean;
  readonly onView: (view: View) => void;
  readonly onClose: () => void;
  readonly onSignout: () => void;
  readonly onShare: () => void;
}
const NAVIGATION: ReadonlyArray<{
  readonly id: View;
  readonly icon: typeof House;
}> = [
  { id: 'overview', icon: House },
  { id: 'flow', icon: ChartNoAxesCombined },
  { id: 'todos', icon: ListChecks },
  { id: 'expenses', icon: Wallet },
  { id: 'goals', icon: Sprout },
  { id: 'targets', icon: Plane },
];

/**
 * Identifies the household and gives the mascot its top-left home.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
function SidebarBrand(props: Pick<SidebarProps, 'members' | 'onView' | 'onClose'>) {
  return (
    <>
      <a
        href="#main"
        className={styles.brand}
        aria-label="Bank Piggy home"
        onClick={() => {
          props.onView('overview');
          props.onClose();
        }}
      >
        <img src="/assets/banker-piggy.png" alt="" />
        <span>
          Bank Piggy<small>Your money, in one place.</small>
        </span>
      </a>
      <div className={styles.household}>
        <span className={styles.householdIcon}>
          <House size={15} />
        </span>
        <span>
          Our household
          <small>{props.members.map((member) => member.name).join(' & ')}</small>
        </span>
      </div>
    </>
  );
}

/**
 * Links only to working finance views.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
function SidebarNavigation(props: Pick<SidebarProps, 'view' | 'onView' | 'onClose'>) {
  return (
    <nav className={styles.nav}>
      {NAVIGATION.map(({ id, icon: Icon }) => (
        <button
          key={id}
          onClick={() => {
            props.onView(id);
            props.onClose();
          }}
          aria-current={props.view === id ? 'page' : undefined}
          className={props.view === id ? styles.active : undefined}
        >
          <Icon size={18} />
          {VIEW_DETAILS[id].label}
        </button>
      ))}
    </nav>
  );
}

/**
 * Names the authenticated member and exposes session sign-out.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
function SidebarUser(props: Pick<SidebarProps, 'session' | 'onSignout'>) {
  return (
    <div className={styles.user}>
      <span className={styles.avatar}>{props.session.viewer?.name.charAt(0) ?? 'P'}</span>
      <div>
        {props.session.viewer?.name ?? 'Your account'}
        <small>Household member</small>
      </div>
      <button aria-label="Sign out" onClick={props.onSignout}>
        <LogOut size={16} />
      </button>
    </div>
  );
}

/**
 * Provides desktop navigation and a keyboard-contained mobile drawer.
 * @param props Component values and event handlers.
 * @returns Rendered interface.
 */
export function Sidebar(props: SidebarProps) {
  const ref = useRef<HTMLElement>(null);
  useDrawerFocus(ref, props.open, props.onClose);
  return (
    <>
      {props.open && (
        <button className={styles.backdrop} onClick={props.onClose} aria-label="Close navigation" />
      )}
      <aside
        ref={ref}
        className={`${styles.sidebar} ${props.open ? styles.open : ''}`}
        aria-label="Main navigation"
      >
        <button
          className={styles.mobileClose}
          onClick={props.onClose}
          aria-label="Close navigation"
        >
          <X size={19} />
        </button>
        <SidebarBrand members={props.members} onView={props.onView} onClose={props.onClose} />
        <SidebarNavigation view={props.view} onView={props.onView} onClose={props.onClose} />
        <button
          className={sharingStyles.shareButton}
          onClick={() => {
            props.onClose();
            props.onShare();
          }}
        >
          Share household
        </button>
        <SidebarUser session={props.session} onSignout={props.onSignout} />
      </aside>
    </>
  );
}
