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

const TEST_TIMED_QUERIES: QueryData[] = [
  createQuery({ queryId: 'fast', queryExecutionDurationMs: 1.5 }),
  createQuery({ queryId: 'slow', queryExecutionDurationMs: 30 }),
];

const renderTable = (queries: QueryData[] = TEST_QUERIES) =>
  render(
    <TestWrapper>
      <OverviewTable
        queries={queries}
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

describe('OverviewTable Elapsed time filter', () => {
  const bound = (name: 'Min' | 'Max') =>
    screen.getByLabelText(name) as HTMLInputElement;

  const type = (name: 'Min' | 'Max', value: string) =>
    fireEvent.change(bound(name), { target: { value } });

  const visibleRows = () =>
    screen
      .queryAllByTestId(/^query-.+-row$/)
      .map((row) => row.getAttribute('data-testid'));

  it('rejects non-numeric input and leaves the rows untouched', async () => {
    renderTable(TEST_TIMED_QUERIES);
    showFilters();

    type('Min', 'abc');

    expect(bound('Min').value).toBe('');
    await waitFor(() =>
      expect(visibleRows()).toEqual(['query-fast-row', 'query-slow-row'])
    );
  });

  it('keeps the last valid value when the new one is not a number', async () => {
    renderTable(TEST_TIMED_QUERIES);
    showFilters();

    type('Min', '1.5');
    await waitFor(() => expect(bound('Min').value).toBe('1.5'));

    type('Min', '1.5x');

    expect(bound('Min').value).toBe('1.5');
  });

  it('filters on a decimal bound', async () => {
    renderTable(TEST_TIMED_QUERIES);
    showFilters();

    type('Min', '1.5');

    // the bound is inclusive, which is what 'timeRangeFilterFn' is there for
    await waitFor(() =>
      expect(visibleRows()).toEqual(['query-fast-row', 'query-slow-row'])
    );

    type('Min', '2');

    await waitFor(() => expect(visibleRows()).toEqual(['query-slow-row']));
  });

  it('rejects non-numeric input in Max and filters on a valid bound', async () => {
    renderTable(TEST_TIMED_QUERIES);
    showFilters();

    type('Max', '10s');

    expect(bound('Max').value).toBe('');

    type('Max', '10');

    await waitFor(() => expect(visibleRows()).toEqual(['query-fast-row']));
  });

  it('does not filter on a lone decimal point', async () => {
    renderTable(TEST_TIMED_QUERIES);
    showFilters();

    type('Min', '.');

    expect(bound('Min').value).toBe('.');
    await waitFor(() =>
      expect(visibleRows()).toEqual(['query-fast-row', 'query-slow-row'])
    );
  });

  it('restores the rows when the bound is emptied', async () => {
    renderTable(TEST_TIMED_QUERIES);
    showFilters();

    type('Min', '2');
    await waitFor(() => expect(visibleRows()).toEqual(['query-slow-row']));

    type('Min', '');

    expect(bound('Min').value).toBe('');
    await waitFor(() =>
      expect(visibleRows()).toEqual(['query-fast-row', 'query-slow-row'])
    );
  });
});
