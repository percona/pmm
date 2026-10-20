import { fireEvent, screen } from '@testing-library/react';
import { hasPeakHub, renderHub } from '../../testUtils';
import { TEST_TREE_COLUMNS, testView } from '../hub.testFixtures';
import type { HubMessages, HubViewModel } from '../hub.types';
import { ClustersSection } from './ClustersSection';

const MESSAGES = {
  name: 'MongoDB',
  unit: { cluster: ['cluster', 'clusters'] },
  empty: { title: 'Start', description: 'Add one', viewer: 'Ask' },
  clusters: { tableLabel: 'MongoDB clusters' },
} as HubMessages;

const renderSection = (view: Partial<HubViewModel>) => {
  const onClearPatches = vi.fn();

  renderHub(
    <ClustersSection
      view={{ ...testView(), ...view }}
      treeColumns={TEST_TREE_COLUMNS}
      messages={MESSAGES}
      onRetry={vi.fn()}
      onShowAllEnvironments={vi.fn()}
      onClearPatches={onClearPatches}
    />
  );

  return onClearPatches;
};

describe('ClustersSection', () => {
  it('shows no filter while every cluster shows', () => {
    renderSection({});

    expect(
      screen.queryByRole('button', { name: /behind on patches/i })
    ).toBeNull();
  });

  it('names the patches filter on a chip that clears it', () => {
    const onClearPatches = renderSection({ patchesFilter: true });

    fireEvent.click(
      screen.getByRole('button', {
        name: 'Clear the filter: behind on patches',
      })
    );
    expect(onClearPatches).toHaveBeenCalled();
    expect(screen.getByRole('heading', { name: 'Clusters' })).toHaveFocus();
    expect(
      screen.getByRole('table', { name: 'MongoDB clusters' })
    ).toBeInTheDocument();
  });

  // EmptyState is not in the published Peak UI yet.
  it.skipIf(!hasPeakHub)(
    'says no cluster is behind instead of an empty table',
    () => {
      const onClearPatches = renderSection({
        patchesFilter: true,
        clusters: [],
      });

      expect(screen.queryByRole('table')).toBeNull();
      expect(
        screen.getByRole('heading', {
          level: 3,
          name: 'No cluster is behind on patches',
        })
      ).toBeInTheDocument();
      fireEvent.click(
        screen.getByRole('button', { name: 'Show all clusters' })
      );
      expect(onClearPatches).toHaveBeenCalled();
    }
  );
});
