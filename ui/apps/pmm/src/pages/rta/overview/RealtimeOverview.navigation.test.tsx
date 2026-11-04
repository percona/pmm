/**
 * Details-pane prev/next must follow the table's filtered rows (navigableQueries),
 * not the full list returned by the API.
 *
 * In the app, OverviewTable syncs navigableQueries from Material React Table after
 * filters are applied. RealtimeOverview uses that list for arrow navigation.
 *
 * We mock OverviewTable here so we can fix navigableQueries (query-1, query-2) while
 * the API still returns an extra query (query-3) that would be next if we used the
 * unfiltered array. RealtimeOverview.test.tsx keeps the real table for other behavior.
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { FC, useEffect } from 'react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { wrapWithQueryProvider } from 'utils/testUtils';
import {
  TEST_MONGO_DB_QUERY_DATA,
  TEST_REAL_TIME_SESSION,
} from 'utils/testStubs';
import { QueryData } from 'types/rta.types';
import RealtimeOverview from './RealtimeOverview';

const queryOne: QueryData = {
  ...TEST_MONGO_DB_QUERY_DATA,
  queryId: 'query-1',
};

const queryTwo: QueryData = {
  ...TEST_MONGO_DB_QUERY_DATA,
  queryId: 'query-2',
};

const queryFilteredOut: QueryData = {
  ...TEST_MONGO_DB_QUERY_DATA,
  queryId: 'query-3',
};

const navigableQueries = [queryOne, queryTwo];

const { searchQueries, getRunningSessions, mockNavigableQueries } = vi.hoisted(
  () => {
    let navigable: QueryData[] = [];

    return {
      searchQueries: vi.fn(),
      getRunningSessions: vi.fn(),
      mockNavigableQueries: {
        get: () => navigable,
        set: (queries: QueryData[]) => {
          navigable = queries;
        },
      },
    };
  }
);

vi.mock('api/rta', () => ({
  searchQueries,
  getRunningSessions,
}));

vi.mock('./table/OverviewTable', () => {
  const MockOverviewTable: FC<{
    onQuerySelected: (query: QueryData) => void;
    onNavigableQueriesChange: (queries: QueryData[]) => void;
  }> = ({ onQuerySelected, onNavigableQueriesChange }) => {
    useEffect(() => {
      onNavigableQueriesChange(mockNavigableQueries.get());
    }, [onNavigableQueriesChange]);

    return (
      <>
        <button
          type="button"
          data-testid="mock-select-first-query"
          onClick={() => onQuerySelected(queryOne)}
        >
          Select first query
        </button>
        <button
          type="button"
          data-testid="mock-drop-selected-from-navigable"
          onClick={() => {
            mockNavigableQueries.set([queryTwo]);
            onNavigableQueriesChange(mockNavigableQueries.get());
          }}
        >
          Drop selected from navigable
        </button>
      </>
    );
  };

  return { default: MockOverviewTable };
});

const renderComponent = () =>
  render(
    wrapWithQueryProvider(
      <MemoryRouter
        initialEntries={[
          `/rta/overview?serviceIds=${TEST_REAL_TIME_SESSION.serviceId}`,
        ]}
      >
        <Routes>
          <Route path="/rta/overview" element={<RealtimeOverview />} />
        </Routes>
      </MemoryRouter>
    )
  );

const openDetailsPaneOnFirstQuery = async () => {
  await waitFor(() =>
    expect(screen.getByTestId('mock-select-first-query')).toBeInTheDocument()
  );
  fireEvent.click(screen.getByTestId('mock-select-first-query'));
  await waitFor(() =>
    expect(screen.getByTestId('query-details-pane')).toHaveAttribute(
      'aria-hidden',
      'false'
    )
  );
};

const getOperationId = () => screen.getByTestId('operation-id-value');

describe('RealtimeOverview details pane navigation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockNavigableQueries.set(navigableQueries);

    searchQueries.mockResolvedValue({
      queries: [...navigableQueries, queryFilteredOut],
    });

    getRunningSessions.mockResolvedValue([TEST_REAL_TIME_SESSION]);
  });

  it('navigates to the next visible query, not the next API query', async () => {
    renderComponent();
    await openDetailsPaneOnFirstQuery();

    expect(getOperationId()).toHaveTextContent('query-1');

    fireEvent.click(screen.getByTestId('details-pane-next-button'));

    await waitFor(() => {
      expect(getOperationId()).toHaveTextContent('query-2');
    });
    expect(getOperationId()).not.toHaveTextContent('query-3');
  });

  it('navigates to the previous visible query', async () => {
    renderComponent();
    await openDetailsPaneOnFirstQuery();

    fireEvent.click(screen.getByTestId('details-pane-next-button'));
    await waitFor(() => {
      expect(getOperationId()).toHaveTextContent('query-2');
    });

    fireEvent.click(screen.getByTestId('details-pane-prev-button'));
    await waitFor(() => {
      expect(getOperationId()).toHaveTextContent('query-1');
    });
  });

  it('disables prev on the first visible query and next on the last', async () => {
    renderComponent();
    await openDetailsPaneOnFirstQuery();

    expect(screen.getByTestId('details-pane-prev-button')).toBeDisabled();
    expect(screen.getByTestId('details-pane-next-button')).not.toBeDisabled();

    fireEvent.click(screen.getByTestId('details-pane-next-button'));

    await waitFor(() => {
      expect(screen.getByTestId('details-pane-prev-button')).not.toBeDisabled();
      expect(screen.getByTestId('details-pane-next-button')).toBeDisabled();
    });
  });

  it('disables navigation when the selected query is not in navigableQueries', async () => {
    renderComponent();
    await openDetailsPaneOnFirstQuery();

    expect(getOperationId()).toHaveTextContent('query-1');

    fireEvent.click(screen.getByTestId('mock-drop-selected-from-navigable'));

    await waitFor(() => {
      expect(screen.getByTestId('details-pane-prev-button')).toBeDisabled();
      expect(screen.getByTestId('details-pane-next-button')).toBeDisabled();
    });

    fireEvent.click(screen.getByTestId('details-pane-next-button'));

    await waitFor(() => {
      expect(getOperationId()).toHaveTextContent('query-1');
    });
  });
});
