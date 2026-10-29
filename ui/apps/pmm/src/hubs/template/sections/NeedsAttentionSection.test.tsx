import { fireEvent, screen, within } from '@testing-library/react';
import { hasPeakHub, renderHub } from '../../testUtils';
import { attentionItem, READY } from '../hub.testFixtures';
import type {
  HubDestination,
  HubNeedsAttentionItem,
  HubSectionState,
} from '../hub.types';
import { attentionSteps, NeedsAttentionSection } from './NeedsAttentionSection';

const ITEMS: HubNeedsAttentionItem[] = [
  attentionItem('billing-eu has no primary and can’t accept writes', {
    severity: 'critical',
    impact: 'Applications writing to billing-eu are failing.',
    location: 'billing-eu',
    alertFiring: true,
    alertService: 'billing-eu-m1',
  }),
  attentionItem('orders-eu shard 2 has no primary', {
    severity: 'critical',
    location: 'orders-eu › orders-eu-shard2',
    since: '15m',
  }),
  attentionItem('9 members are down in 6 other EU clusters', {
    severity: 'error',
    location: 'cart-eu, users-eu and 4 more',
    since: 'up to 18m',
    issueCount: 9,
    alertFiring: true,
    destination: {
      label: 'Show affected clusters',
      route: '#clusters',
      kind: 'inpage',
    },
  }),
  attentionItem('3 nightly backups failed', {
    severity: 'error',
    since: '8h ago',
    issueCount: 3,
    destination: {
      label: 'MongoDB backups',
      route: '/graph/backup/inventory',
      kind: 'page',
    },
  }),
  attentionItem('PMM gets no data from 4 clusters', {
    environment: 'prod, staging',
    issueCount: 4,
  }),
  attentionItem('4 queries are running long', { issueCount: 4 }),
  attentionItem('Upgrade notes', {
    destination: {
      label: 'Release notes',
      route: 'https://docs.percona.com',
      kind: 'external',
    },
  }),
];

const LOOK = (destination: HubDestination) => ({
  path: ['Apps', destination.label],
});

const renderSection = ({
  items = ITEMS,
  state = READY,
  revealed = 0,
  environmentFiltered = false,
  onRevealedChange = vi.fn(),
}: {
  items?: HubNeedsAttentionItem[];
  state?: HubSectionState;
  revealed?: number;
  environmentFiltered?: boolean;
  onRevealedChange?: (revealed: number) => void;
} = {}) =>
  renderHub(
    <NeedsAttentionSection
      items={items}
      state={state}
      environmentFiltered={environmentFiltered}
      describeDestination={LOOK}
      revealed={revealed}
      onRevealedChange={onRevealedChange}
      onRetry={vi.fn()}
    />,
    { path: '/mongodb?hubState=unhealthy' }
  );

const rows = () => screen.getAllByTestId('attention-row');
const row = (title: string) =>
  rows().find((item) => item.textContent?.includes(title)) as HTMLElement;

describe.skipIf(!hasPeakHub)('NeedsAttentionSection', () => {
  it('is not there at all when nothing needs attention', () => {
    renderSection({ items: [] });

    expect(screen.queryByRole('region')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('heading', { name: /Needs attention/ })
    ).not.toBeInTheDocument();
  });

  it('counts the issues behind the rows in its heading, and has no Alerts link', () => {
    renderSection();

    const section = screen.getByRole('region', { name: 'Needs attention' });

    expect(section).toHaveTextContent('Needs attention23 issues');
    expect(within(section).getByText('issues')).toBeVisible();
    expect(
      within(section).queryByRole('link', { name: 'Alerts' })
    ).not.toBeInTheDocument();
  });

  it('shows five rows and folds the rest behind Show N more', () => {
    const onRevealedChange = vi.fn();

    renderSection({ onRevealedChange });

    const more = screen.getByRole('button', { name: 'Show 2 more' });

    expect(rows()).toHaveLength(5);
    expect(more).toHaveAttribute('aria-expanded', 'false');
    fireEvent.click(more);
    expect(onRevealedChange).toHaveBeenCalledWith(1);
  });

  it('reads Show fewer once every row shows, and folds back to five', () => {
    const onRevealedChange = vi.fn();

    renderSection({ revealed: Number.POSITIVE_INFINITY, onRevealedChange });

    const fewer = screen.getByRole('button', { name: 'Show fewer' });

    expect(rows()).toHaveLength(7);
    expect(fewer).toHaveAttribute('aria-expanded', 'true');
    fireEvent.click(fewer);
    expect(onRevealedChange).toHaveBeenCalledWith(0);
  });

  it('reveals five rows a step, the last step up to seven', () => {
    const many = Array.from({ length: 22 }, (_, index) =>
      attentionItem(`issue ${index + 1}`)
    );

    expect(attentionSteps(22)).toEqual([5, 5, 7]);
    expect(attentionSteps(12)).toEqual([7]);
    expect(attentionSteps(13)).toEqual([5, 3]);
    expect(attentionSteps(5)).toEqual([]);

    renderSection({ items: many, revealed: 1 });

    expect(rows()).toHaveLength(10);
    expect(screen.getByRole('button', { name: 'Show 5 more' })).toBeVisible();
  });

  it('names the severity in hidden text, with the State indicator’s icon', () => {
    renderSection();

    const critical = row('billing-eu has no primary');

    expect(within(critical).getByText('Critical:')).toBeInTheDocument();
    expect(
      critical.querySelector('svg[data-severity="critical"]')
    ).toBeTruthy();
    expect(
      row('nightly backups').querySelector('svg[data-severity="error"]')
    ).toBeTruthy();
  });

  it('shows the location only when the title does not already name it', () => {
    renderSection();

    expect(row('billing-eu has no primary')).not.toHaveTextContent(
      'billing-eu ·'
    );
    expect(row('orders-eu shard 2')).toHaveTextContent(
      'orders-eu › orders-eu-shard2 · prod · for 15m'
    );
  });

  it('says how long a condition holds, and when an event happened', () => {
    renderSection();

    expect(row('9 members are down')).toHaveTextContent('for up to 18m');
    expect(row('nightly backups')).toHaveTextContent('· 8h ago');
    expect(row('nightly backups')).not.toHaveTextContent('for 8h ago');
  });

  it('leaves the environment out while one environment is selected', () => {
    renderSection({ environmentFiltered: true });

    expect(row('orders-eu shard 2')).toHaveTextContent(
      'orders-eu › orders-eu-shard2 · for 15m'
    );
  });

  it('links a firing alert to Alerts, filtered to its service when it has one', () => {
    renderSection();

    expect(
      within(row('billing-eu has no primary')).getByRole('link', {
        name: 'Alert firing',
      })
    ).toHaveAttribute('href', '/alerting/status?f.serviceName=billing-eu-m1');
    expect(
      within(row('9 members are down')).getByRole('link', {
        name: 'Alert firing',
      })
    ).toHaveAttribute('href', '/alerting/status');
  });

  it('gives each row one action, its destination with its params, a section of this page keeping the hub’s', () => {
    renderSection({ revealed: Number.POSITIVE_INFINITY });

    expect(
      within(row('billing-eu has no primary')).getByRole('link', {
        name: /ReplSet summary/,
      })
    ).toHaveAttribute(
      'href',
      '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-cluster=orders'
    );
    expect(
      within(row('9 members are down')).getByRole('link', {
        name: /Show affected clusters/,
      })
    ).toHaveAttribute('href', '/mongodb?hubState=unhealthy#clusters');
    expect(
      within(row('Upgrade notes')).getByRole('link', { name: /Release notes/ })
    ).toHaveAttribute('target', '_blank');
  });

  it('shows where the action goes in the sidebar on hover', async () => {
    renderSection();

    fireEvent.mouseEnter(
      within(row('nightly backups')).getByText('MongoDB backups')
    );

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Apps › MongoDB backups'
    );
  });

  it('keeps each line to one line, showing the full impact in a tooltip while it is cut', async () => {
    renderSection();

    const impact = within(row('3 nightly backups failed')).getByText(
      'Reads from it may return stale data.'
    );

    expect(impact).toHaveStyle({ whiteSpace: 'nowrap' });
    Object.defineProperty(impact, 'scrollWidth', { value: 400 });
    Object.defineProperty(impact, 'clientWidth', { value: 200 });
    fireEvent.mouseEnter(impact);

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Reads from it may return stale data.'
    );
  });

  it('passes a click on the severity icon to the row’s action', () => {
    renderSection();

    const action = within(row('nightly backups')).getByRole('link', {
      name: /MongoDB backups/,
    });
    const click = vi.fn((event: MouseEvent) => event.preventDefault());

    action.addEventListener('click', click);
    fireEvent.click(
      row('nightly backups').querySelector('svg[data-severity]') as Element
    );

    expect(click).toHaveBeenCalled();
  });

  it('keeps its rows under the warning when only a supplementary source failed', () => {
    renderSection({
      state: {
        status: 'error',
        source: 'alerts',
        message: 'Firing alerts could not be read, so none is listed here.',
        partial: true,
      },
    });

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Firing alerts could not be read'
    );
    expect(rows().length).toBeGreaterThan(0);
  });
});
