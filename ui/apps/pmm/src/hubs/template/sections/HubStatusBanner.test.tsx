import { describe } from 'vitest';
import { screen, within } from '@testing-library/react';
import { hasPeakHub, renderHub } from '../../testUtils';
import { testView } from '../hub.testFixtures';
import type { HubStatus } from '../hub.types';
import { HubStatusBanner } from './HubStatusBanner';

const renderBanner = (status: Partial<HubStatus> = {}) =>
  renderHub(
    <HubStatusBanner
      status={{ ...testView().status, ...status }}
      name="MongoDB"
      unit={['cluster', 'clusters']}
      loading={false}
      matchesNothing={false}
    />
  );

const stateOf = (text: string) =>
  screen.getByText(text).closest('[data-state]')?.getAttribute('data-state');

describe.skipIf(!hasPeakHub)('HubStatusBanner', () => {
  it('lists cluster health, alerts and backups under the hub’s status', () => {
    renderBanner();

    const banner = screen.getByRole('list', { name: 'MongoDB status' });

    expect(within(banner).getAllByRole('listitem')).toHaveLength(3);
    expect(stateOf('2 of 3 clusters need attention')).toBe('mixedProblems');
    expect(stateOf('2 alerts firing')).toBe('errors');
    expect(stateOf('Backups on schedule, last 6h ago')).toBe('default');
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
