import { fireEvent, screen, within } from '@testing-library/react';
import { renderHub } from '../../testUtils';
import { READY, testDoor, testView } from '../hub.testFixtures';
import type { HubQueries, HubSectionState } from '../hub.types';
import { QueriesSection } from './QueriesSection';

const QUERIES = testView().queries;

const SLOWER: HubQueries['slower'] = [
  {
    id: 'q1',
    text: 'db.orders.find({status: ?, createdAt: ?})',
    headline: '6× slower than usual',
    detail: '0.84s avg · usual 0.14s · 1.2k calls in 1h',
    severity: 'warning',
    door: testDoor('Open', '/graph/d/pmm-qan?filter_by=q1', {
      context: 'in Query analytics',
    }),
  },
];

const RUNNING: HubQueries['runningLong'] = [
  {
    id: 'r1',
    text: 'aggregate on shop.orders',
    headline: 'running for 14m',
    detail: 'orders-shard1-m1',
    severity: 'warning',
    door: testDoor('Open', '/rta/overview?serviceIds=svc-1', {
      context: 'in Real-time queries',
    }),
  },
];

const renderSection = (
  queries: Partial<HubQueries> = {},
  state: HubSectionState = READY,
  empty = false
) =>
  renderHub(
    <QueriesSection
      queries={{ ...QUERIES, ...queries }}
      state={state}
      empty={empty}
      onRetry={vi.fn()}
    />
  );

describe('QueriesSection', () => {
  it.each<[string, HubSectionState]>([
    ['ready', READY],
    ['loading', { status: 'loading' }],
    ['failed', { status: 'error' }],
  ])('keeps its two links while %s', (_, state) => {
    renderSection({}, state);

    expect(
      screen.getByRole('link', { name: 'Query analytics' })
    ).toHaveAttribute('href', '/graph/d/pmm-qan');
    expect(
      screen.getByRole('link', { name: 'Real-time queries' })
    ).toHaveAttribute('href', '/rta');
  });

  it('says so in one sentence when no query is off', () => {
    renderSection();

    expect(
      screen.getByText('No query is slower than usual or running long.')
    ).toBeInTheDocument();
    expect(screen.queryByRole('list')).not.toBeInTheDocument();
  });

  it('vouches for no query when nothing is monitored, keeping the links', () => {
    renderSection({}, READY, true);

    expect(
      screen.queryByText('No query is slower than usual or running long.')
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: 'Query analytics' })
    ).toBeInTheDocument();
  });

  it('groups the queries that are off under named lists', async () => {
    renderSection({ slower: SLOWER, runningLong: RUNNING });

    const slower = screen.getByRole('list', { name: 'Slower than usual' });
    const running = screen.getByRole('list', { name: 'Running long now' });

    expect(
      within(slower).getByText('6× slower than usual')
    ).toBeInTheDocument();
    expect(
      within(slower).getByRole('link', { name: 'Open in Query analytics' })
    ).toHaveAttribute('href', '/graph/d/pmm-qan?filter_by=q1');
    expect(
      within(running).getByRole('link', { name: 'Open in Real-time queries' })
    ).toBeInTheDocument();

    const text = within(slower).getByText(
      'db.orders.find({status: ?, createdAt: ?})'
    );

    fireEvent.mouseOver(text);

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'db.orders.find({status: ?, createdAt: ?})'
    );
  });
});
