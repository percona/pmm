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
  ])(
    'leaves the ways into query data to the hub’s tiles while %s',
    (_, state) => {
      renderSection({}, state);

      expect(
        screen.getByRole('heading', { name: 'Queries' })
      ).toBeInTheDocument();
      expect(screen.queryByRole('link')).not.toBeInTheDocument();
    }
  );

  it('says so in one sentence when no query is off', () => {
    renderSection();

    expect(
      screen.getByText('No query is slower than usual.')
    ).toBeInTheDocument();
    expect(screen.queryByRole('list')).not.toBeInTheDocument();
  });

  it('vouches for no query when nothing is monitored', () => {
    renderSection({}, READY, true);

    expect(
      screen.queryByText('No query is slower than usual.')
    ).not.toBeInTheDocument();
  });

  it('lists the queries slower than usual, leaving those running long to Needs attention', async () => {
    renderSection({ slower: SLOWER });

    const slower = screen.getByRole('list', { name: 'Slower than usual' });

    expect(
      screen.queryByRole('list', { name: 'Running long now' })
    ).not.toBeInTheDocument();

    expect(
      within(slower).getByText('6× slower than usual')
    ).toBeInTheDocument();
    expect(
      within(slower).getByRole('link', { name: 'Open in Query analytics' })
    ).toHaveAttribute('href', '/graph/d/pmm-qan?filter_by=q1');

    const text = within(slower).getByText(
      'db.orders.find({status: ?, createdAt: ?})'
    );

    fireEvent.mouseOver(text);

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'db.orders.find({status: ?, createdAt: ?})'
    );
  });
});
