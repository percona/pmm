import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { TEST_MONGO_DB_QUERY_DATA } from 'utils/testStubs';
import type { QueryData } from 'types/rta.types';
import OverviewTable from './OverviewTable';

const createQuery = (overrides: Partial<QueryData>): QueryData => ({
  ...TEST_MONGO_DB_QUERY_DATA,
  ...overrides,
});

// 's1' is a substring of 'mongodb-rs101' only, but appears as a subsequence in
// all three hosts, so a fuzzy match would leave every row visible.
const TEST_QUERIES: QueryData[] = [
  createQuery({ queryId: 'query-1', serviceName: 'mongodb-rs101' }),
  createQuery({ queryId: 'query-2', serviceName: 'mongodb-rs201' }),
  createQuery({ queryId: 'query-3', serviceName: 'postgres-1' }),
];

const renderTable = () =>
  render(
    <TestWrapper>
      <OverviewTable
        queries={TEST_QUERIES}
        onQuerySelected={vi.fn()}
        onNavigableQueriesChange={vi.fn()}
      />
    </TestWrapper>
  );

const showFilters = () =>
  fireEvent.click(screen.getByRole('button', { name: /show\/hide filters/i }));

const filterByHost = (value: string) =>
  fireEvent.change(screen.getByLabelText(/filter by host/i), {
    target: { value },
  });

describe('OverviewTable Host filter', () => {
  it('keeps only the rows whose host contains the filter value', async () => {
    renderTable();
    showFilters();

    expect(screen.getByTestId('query-query-1-row')).toBeInTheDocument();
    expect(screen.getByTestId('query-query-2-row')).toBeInTheDocument();
    expect(screen.getByTestId('query-query-3-row')).toBeInTheDocument();

    filterByHost('s1');

    await waitFor(() =>
      expect(screen.queryByTestId('query-query-2-row')).toBeNull()
    );
    expect(screen.queryByTestId('query-query-3-row')).toBeNull();
    expect(screen.getByTestId('query-query-1-row')).toBeInTheDocument();
  });

  it('matches the host case-insensitively', async () => {
    renderTable();
    showFilters();

    filterByHost('MONGODB-RS101');

    await waitFor(() =>
      expect(screen.queryByTestId('query-query-2-row')).toBeNull()
    );
    expect(screen.queryByTestId('query-query-3-row')).toBeNull();
    expect(screen.getByTestId('query-query-1-row')).toBeInTheDocument();
  });
});
