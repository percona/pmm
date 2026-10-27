import { fireEvent, screen, within } from '@testing-library/react';
import { hasPeakHub, renderHub } from '../../testUtils';
import { READY } from '../hub.testFixtures';
import type { HubRecommendation } from '../hub.types';
import { RecommendationsSection } from './RecommendationsSection';

const item = (
  title: string,
  overrides: Partial<HubRecommendation> = {}
): HubRecommendation => ({
  kind: 'info',
  title,
  impact: 'They can only be restored to their last scheduled backup.',
  location: 'orders-us, cart-us and 7 more',
  clusterCount: 9,
  source: 'Backups',
  destination: {
    label: 'Turn on in Scheduled backups',
    route: '/graph/backup/scheduled',
    kind: 'page',
  },
  ...overrides,
});

const ITEMS = [
  item('4 clusters have no backups', { kind: 'notset' }),
  item('Point-in-time recovery is off on 9 production clusters', {
    environment: 'prod',
  }),
  item('Advisors flagged 7 clusters for security settings'),
  item('14 clusters are behind on patches'),
  item('6 clusters run MongoDB 6.0', {
    destination: {
      label: 'Upgrade guide',
      route: 'https://docs.percona.com/upgrade',
      kind: 'external',
    },
  }),
];

const renderSection = ({
  items = ITEMS,
  visibleCount = 3,
  expanded = false,
  environmentFiltered = false,
  onExpandedChange = vi.fn(),
} = {}) =>
  renderHub(
    <RecommendationsSection
      items={items}
      state={READY}
      environmentFiltered={environmentFiltered}
      describeDestination={({ label }) => ({ path: ['Apps', label] })}
      visibleCount={visibleCount}
      wide={false}
      expanded={expanded}
      onExpandedChange={onExpandedChange}
      onRetry={vi.fn()}
    />
  );

const rows = () => screen.getAllByTestId('recommendation-row');

describe.skipIf(!hasPeakHub)('RecommendationsSection', () => {
  it('is not there at all without recommendations', () => {
    renderSection({ items: [] });

    expect(screen.queryByRole('region')).not.toBeInTheDocument();
  });

  it('heads its rows with the title alone, no count', () => {
    renderSection();

    expect(
      screen.getByRole('heading', { level: 2 }).parentElement
    ).toHaveTextContent(/^Recommendations$/);
  });

  it('shows the given number of rows and folds the rest behind Show N more', () => {
    const onExpandedChange = vi.fn();

    renderSection({ onExpandedChange });

    expect(rows()).toHaveLength(3);
    fireEvent.click(screen.getByRole('button', { name: 'Show 2 more' }));
    expect(onExpandedChange).toHaveBeenCalledWith(true);
  });

  it('reads Show fewer once open', () => {
    renderSection({ expanded: true });

    expect(rows()).toHaveLength(5);
    expect(screen.getByRole('button', { name: 'Show fewer' })).toHaveAttribute(
      'aria-expanded',
      'true'
    );
  });

  it('marks each kind with its icon and hidden label, in a quieter title', () => {
    renderSection();

    const [notSet, info] = rows();

    expect(notSet.querySelector('svg[data-severity="notSetUp"]')).toBeTruthy();
    expect(within(notSet).getByText('Not set up:')).toBeInTheDocument();
    expect(info.querySelector('svg[data-severity="info"]')).toBeTruthy();
    expect(within(info).getByText('Info:')).toBeInTheDocument();
  });

  it('meta line: the clusters, then the environment when they share one', () => {
    renderSection();

    expect(rows()[0]).toHaveTextContent('orders-us, cart-us and 7 more');
    expect(rows()[0]).not.toHaveTextContent(' · ');
    expect(rows()[1]).toHaveTextContent('orders-us, cart-us and 7 more · prod');
  });

  it('leaves the environment out while one environment is selected', () => {
    renderSection({ environmentFiltered: true });

    expect(rows()[1]).not.toHaveTextContent('· prod');
  });

  it('gives each row one action, a link out opening a new tab', () => {
    renderSection({ expanded: true });

    expect(
      within(rows()[1]).getByRole('link', {
        name: /Turn on in Scheduled backups/,
      })
    ).toHaveAttribute('href', '/graph/backup/scheduled');
    expect(
      within(rows()[4]).getByRole('link', { name: /Upgrade guide/ })
    ).toHaveAttribute('target', '_blank');
  });
});
