import { screen, within } from '@testing-library/react';
import { renderHub } from '../../testUtils';
import { minutesAgo, READY, testDoor, testView } from '../hub.testFixtures';
import type { HubDashboardLink, HubDashboards, HubDoor } from '../hub.types';
import { DashboardsSection } from './DashboardsSection';

const link = (
  uid: string,
  label: string,
  extra: Partial<HubDashboardLink> = {}
): HubDashboardLink => ({ uid, label, to: `/graph/d/${uid}`, ...extra });

const renderSection = (
  dashboards: Partial<HubDashboards> = {},
  allApps?: HubDoor
) =>
  renderHub(
    <DashboardsSection
      dashboards={{ ...testView().dashboards, ...dashboards }}
      allApps={allApps}
      state={READY}
      onRetry={vi.fn()}
    />
  );

describe('DashboardsSection', () => {
  it('lists recommended, developer and recent dashboards as named groups', () => {
    renderSection({
      recommended: [
        link('mongodb-instance-overview', 'Instances overview', {
          context: 'for prod',
        }),
      ],
      forDevelopers: [link('mongodb-unused-indexes', 'Unused indexes')],
      recent: [
        link('mongodb-replicaset-summary', 'ReplSet summary', {
          visitedAt: minutesAgo(5),
        }),
      ],
    });

    const recommended = screen.getByRole('list', {
      name: 'Recommended for what’s running',
    });

    expect(
      within(recommended).getByRole('link', {
        name: 'Instances overview for prod',
      })
    ).toHaveAttribute('href', '/graph/d/mongodb-instance-overview');
    expect(
      within(screen.getByRole('list', { name: 'For developers' })).getByRole(
        'link',
        { name: 'Unused indexes' }
      )
    ).toBeInTheDocument();
    expect(
      within(screen.getByRole('list', { name: 'Recently visited' })).getByRole(
        'link',
        { name: /^ReplSet summary/ }
      )
    ).toHaveTextContent('5m ago');
  });

  it('says where recent dashboards will appear while there are none', () => {
    renderSection();

    expect(
      within(screen.getByRole('list', { name: 'Recently visited' })).getByText(
        'Dashboards you open appear here.'
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('list', { name: 'For developers' })
    ).not.toBeInTheDocument();
  });

  it('opens every dashboard of the technology, with the count', () => {
    renderSection({
      all: {
        label: 'All MongoDB dashboards',
        to: '/graph/dashboards/f/mongodb-folder/',
        count: 13,
      },
    });

    expect(
      screen.getByRole('link', { name: 'All MongoDB dashboards 13' })
    ).toHaveAttribute('href', '/graph/dashboards/f/mongodb-folder/');
  });

  it('opens every app after its own door, so the apps without a tile stay a click away', () => {
    renderSection({}, testDoor('All apps', '/prototype/all-apps'));

    const heading = screen.getByRole('heading', { name: 'Dashboards' })
      .parentElement?.parentElement as HTMLElement;

    expect(
      within(heading)
        .getAllByRole('link')
        .map((door) => door.textContent)
    ).toEqual(['All MongoDB dashboards', 'All apps']);
    expect(screen.getByRole('link', { name: 'All apps' })).toHaveAttribute(
      'href',
      '/prototype/all-apps'
    );
  });

  it('opens no apps door when it has none', () => {
    renderSection({}, undefined);

    expect(
      screen.queryByRole('link', { name: 'All apps' })
    ).not.toBeInTheDocument();
  });
});
