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

const renderSection = (
  view: Partial<HubViewModel>,
  patches?: { behind: number; unknown: number }
) => {
  const onShowPatches = vi.fn();
  const onClearPatches = vi.fn();
  const base = testView();

  renderHub(
    <ClustersSection
      view={{
        ...base,
        ...view,
        status: { ...base.status, ...(patches ? { patches } : {}) },
      }}
      treeColumns={TEST_TREE_COLUMNS}
      messages={MESSAGES}
      onRetry={vi.fn()}
      onShowAllEnvironments={vi.fn()}
      onShowPatches={onShowPatches}
      onClearPatches={onClearPatches}
    />
  );

  return { onShowPatches, onClearPatches };
};

describe('ClustersSection', () => {
  it('offers no patches filter while no cluster is behind', () => {
    renderSection({});

    expect(
      screen.queryByRole('button', { name: /behind on patches/i })
    ).toBeNull();
  });

  it('filters to the clusters behind on patches with a toggle chip', () => {
    const { onShowPatches } = renderSection({}, { behind: 2, unknown: 0 });
    const chip = screen.getByRole('button', { name: '2 behind on patches' });

    expect(chip).toHaveAttribute('aria-pressed', 'false');
    fireEvent.click(chip);
    expect(onShowPatches).toHaveBeenCalled();
  });

  it('clears the filter from the same chip, which keeps focus', () => {
    const { onClearPatches } = renderSection(
      { patchesFilter: true },
      { behind: 2, unknown: 0 }
    );
    const chip = screen.getByRole('button', { name: '2 behind on patches' });

    expect(chip).toHaveAttribute('aria-pressed', 'true');
    chip.focus();
    fireEvent.click(chip);
    expect(onClearPatches).toHaveBeenCalled();
    expect(chip).toHaveFocus();
    expect(
      screen.getByRole('table', { name: 'MongoDB clusters' })
    ).toBeInTheDocument();
  });

  it('moves focus to the heading when clearing removes the chip', () => {
    const { onClearPatches } = renderSection({ patchesFilter: true });

    fireEvent.click(screen.getByRole('button', { name: 'Behind on patches' }));
    expect(onClearPatches).toHaveBeenCalled();
    expect(screen.getByRole('heading', { name: 'Clusters' })).toHaveFocus();
  });

  // EmptyState is not in the published Peak UI yet.
  it.skipIf(!hasPeakHub)(
    'says no cluster is behind instead of an empty table',
    () => {
      const { onClearPatches } = renderSection({
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
