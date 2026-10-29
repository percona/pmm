import { fireEvent, screen, within } from '@testing-library/react';
import { hasPeakHub, renderHub } from '../../testUtils';
import type { HubDestination } from '../hub.types';
import { HubShortcuts } from './HubShortcuts';

const SHORTCUTS: HubDestination[] = [
  { label: 'Clusters', route: '/mongodb/clusters', kind: 'page' },
  {
    label: 'Query analytics',
    route: '/graph/d/pmm-qan/pmm-query-analytics',
    params: { 'var-service_type': 'mongodb', 'var-environment': 'prod' },
    kind: 'page',
  },
  { label: 'Fleet', route: '/operations', kind: 'page' },
];

const PATHS: Record<string, string[]> = {
  Clusters: ['MongoDB', 'Clusters'],
  'Query analytics': ['Query analytics'],
  Fleet: ['Apps', 'Operations for MongoDB', 'Fleet'],
};

const renderShortcuts = () =>
  renderHub(
    <HubShortcuts
      shortcuts={SHORTCUTS}
      describeDestination={({ label }) => ({ path: PATHS[label] })}
    />
  );

describe.skipIf(!hasPeakHub)('HubShortcuts', () => {
  it('lists a tile per place, in order, in a Shortcuts landmark', () => {
    renderShortcuts();

    const nav = screen.getByRole('navigation', { name: 'Shortcuts' });

    expect(
      within(nav)
        .getAllByRole('link')
        .map((link) => [link.textContent, link.getAttribute('href')])
    ).toEqual([
      ['Clusters', '/mongodb/clusters'],
      [
        'Query analytics',
        '/graph/d/pmm-qan/pmm-query-analytics?var-service_type=mongodb&var-environment=prod',
      ],
      ['Fleet', '/operations'],
    ]);
  });

  it('says where each place sits in the sidebar in its tooltip', async () => {
    renderShortcuts();

    fireEvent.mouseOver(screen.getByRole('link', { name: 'Fleet' }));

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Apps › Operations for MongoDB › Fleet'
    );
  });

  it('rules the tiles as a grid, leaving its top to the header’s divider', () => {
    renderShortcuts();

    const grid = getComputedStyle(screen.getByRole('list'));

    expect([grid.borderTopWidth, grid.borderLeftWidth]).toEqual(['0px', '1px']);
  });
});
