import { describe } from 'vitest';
import { fireEvent, screen, within } from '@testing-library/react';
import { hasPeakHub, renderHub } from '../../testUtils';
import { testView } from '../hub.testFixtures';
import type { HubStatus } from '../hub.types';
import { HubStatusBanner } from './HubStatusBanner';

const renderBanner = (
  status: Partial<HubStatus> = {},
  { issues = 0, path }: { issues?: number; path?: string } = {}
) =>
  renderHub(
    <HubStatusBanner
      status={{ ...testView().status, ...status }}
      name="MongoDB"
      unit={['cluster', 'clusters']}
      loading={false}
      matchesNothing={false}
      issues={issues}
    />,
    { path }
  );

const stateOf = (text: string) =>
  screen.getByText(text).closest('[data-state]')?.getAttribute('data-state');

const location = () => screen.getByTestId('hub-location');

describe.skipIf(!hasPeakHub)('HubStatusBanner', () => {
  it('lists cluster health, alerts and backups under the hub’s status', () => {
    renderBanner();

    const banner = screen.getByRole('list', { name: 'MongoDB status' });

    expect(within(banner).getAllByRole('listitem')).toHaveLength(3);
    expect(stateOf('2 of 3 clusters need attention')).toBe('mixedProblems');
    expect(stateOf('2 alerts firing')).toBe('errors');
    expect(stateOf('Backups on schedule, last 6h ago')).toBe('default');
  });

  it('puts a facet with an action on a card that takes it from anywhere', () => {
    renderBanner();

    const [, alerts, backups] = screen.getAllByRole('listitem');
    const open = within(alerts).getByRole('link', { name: 'Open alerts' });

    expect(open).toHaveAttribute('href', '/alerting/status');
    expect(within(open).getByTestId('ArrowForwardIcon')).toBeInTheDocument();
    expect(alerts.querySelector('.MuiCard-root')).not.toBeNull();
    expect(backups.querySelector('.MuiCard-root')).toBeNull();

    fireEvent.click(screen.getByText('2 alerts firing'));

    expect(location()).toHaveTextContent('/alerting/status');
  });

  it('reviews cluster problems down the page, keeping the hub’s params', () => {
    renderBanner({}, { issues: 4, path: '/mongodb?hubState=unhealthy' });

    const review = screen.getByRole('link', { name: 'Review 4 issues' });

    expect(review).toHaveAttribute(
      'href',
      '/mongodb?hubState=unhealthy#needs-attention'
    );
    expect(within(review).getByTestId('ArrowDownwardIcon')).toBeInTheDocument();

    fireEvent.click(screen.getByText('2 of 3 clusters need attention'));

    expect(location()).toHaveTextContent(
      '/mongodb?hubState=unhealthy#needs-attention'
    );
  });

  it('links each missing piece to where it is set up, inside its facet', () => {
    renderBanner({
      alertRules: 'none',
      alerts: 0,
      backupCoverage: { protected: 2, protectable: 3 },
    });

    const [, alerts, backups] = screen.getAllByRole('listitem');

    expect(
      within(alerts).getByRole('link', { name: 'Set up alerts' })
    ).toHaveAttribute('href', '/graph/alerting/alert-rule-templates');
    expect(
      within(backups).getByRole('link', { name: 'Set up backups' })
    ).toHaveAttribute('href', '/graph/backup/new');
    expect(stateOf('No alerts set up')).toBe('notSetUp');
    expect(stateOf('2 of 3 clusters protected')).toBe('notSetUp');
  });
});
