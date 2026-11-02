import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { useEffect } from 'react';
import { setSearchExperiment } from '../search.experiment';
import { SEARCH_DIALOG_TEST_ID } from '../search.constants';
import {
  closeSearch,
  getSearchState,
  openSearch,
  resetSearchStore,
} from '../search.store';
import {
  mockSearchApis,
  renderWithSearchContexts,
  setPlatform,
} from '../search.testFixtures';
import { enableSearchForTest, hasPeakSearch } from '../testUtils';
import { SearchProvider } from './SearchProvider';
import type * as SearchRuntimeModule from './SearchRuntime';

const runtime = vi.hoisted(() => ({ real: false, gateRenders: 0 }));

// The published Peak UI has no SearchDialog, so the runtime is stubbed unless a suite asks for the real one.
vi.mock('./SearchRuntime', async (importOriginal) => {
  const actual = await importOriginal<typeof SearchRuntimeModule>();

  return {
    ...actual,
    SearchRuntimeGate: () => {
      runtime.gateRenders += 1;

      return runtime.real ? <actual.SearchRuntimeGate /> : null;
    },
  };
});

describe('SearchProvider', () => {
  beforeEach(() => {
    runtime.gateRenders = 0;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    setSearchExperiment(false);
    resetSearchStore();
  });

  it('renders only its children while search is off', () => {
    const addEventListener = vi.spyOn(window, 'addEventListener');

    const { container } = render(
      <SearchProvider>
        <p>Page</p>
      </SearchProvider>
    );

    expect(container.innerHTML).toBe('<p>Page</p>');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(
      addEventListener.mock.calls.filter(
        ([type]) => type === 'keydown' || type === 'message'
      )
    ).toEqual([]);
    expect(runtime.gateRenders).toBe(0);
  });

  it('keeps its children mounted when search turns on', () => {
    const mounts = vi.fn();
    const Child = () => {
      useEffect(() => mounts(), []);

      return <p>Page</p>;
    };

    render(
      <SearchProvider>
        <Child />
      </SearchProvider>
    );
    const node = screen.getByText('Page');

    act(() => setSearchExperiment(true));

    expect(screen.getByText('Page')).toBe(node);
    expect(mounts).toHaveBeenCalledTimes(1);
    expect(runtime.gateRenders).toBeGreaterThan(0);
  });

  it('clears the search state when search turns off', () => {
    setSearchExperiment(true);
    render(
      <SearchProvider>
        <p>Page</p>
      </SearchProvider>
    );

    act(() => setSearchExperiment(false));

    expect(getSearchState()).toEqual({ open: false, query: '', source: null });
  });

  describe.skipIf(!hasPeakSearch)('with Peak UI search, search on', () => {
    enableSearchForTest();

    beforeEach(() => {
      runtime.real = true;
      setPlatform('MacIntel');
      mockSearchApis();
    });

    afterEach(() => {
      runtime.real = false;
    });

    const setup = () =>
      renderWithSearchContexts(
        <SearchProvider>
          <button type="button">Opener</button>
        </SearchProvider>
      );

    it('opens with ⌘K, closes with ⌘K, and returns focus after Esc', async () => {
      setup();
      const opener = screen.getByRole('button', { name: 'Opener' });
      opener.focus();

      fireEvent.keyDown(opener, { key: 'k', metaKey: true });
      expect(
        await screen.findByRole('dialog', { name: 'Search PMM' })
      ).toBeInTheDocument();

      fireEvent.keyDown(screen.getByRole('combobox'), {
        key: 'k',
        metaKey: true,
      });
      await waitFor(() =>
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      );

      opener.focus();
      fireEvent.keyDown(opener, { key: 'k', metaKey: true });
      fireEvent.keyDown(await screen.findByRole('combobox'), {
        key: 'Escape',
      });

      await waitFor(() => expect(document.activeElement).toBe(opener));
    });

    it('opens when Grafana asks for it', async () => {
      setup();

      act(() => {
        window.dispatchEvent(
          new MessageEvent('message', {
            data: { type: 'OPEN_SEARCH', source: 'GRAFANA' },
          })
        );
      });

      expect(
        await screen.findByRole('dialog', { name: 'Search PMM' })
      ).toBeInTheDocument();
      expect(getSearchState().source).toBe('grafana');
    });

    it('reopens for Grafana while its own dialog is still fading out', async () => {
      setup();
      act(() => openSearch('sidebar'));
      await screen.findByRole('dialog', { name: 'Search PMM' });

      act(() => closeSearch());
      expect(screen.getByTestId(SEARCH_DIALOG_TEST_ID)).toBeInTheDocument();
      act(() => {
        window.dispatchEvent(
          new MessageEvent('message', {
            data: { type: 'OPEN_SEARCH', source: 'GRAFANA' },
          })
        );
      });

      expect(getSearchState()).toMatchObject({ open: true, source: 'grafana' });
    });
  });
});
