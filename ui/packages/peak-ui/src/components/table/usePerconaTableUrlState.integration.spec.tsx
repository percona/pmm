import { act, render, screen, waitFor } from '@testing-library/react';
import { type MRT_ColumnDef } from 'material-react-table';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { baseThemeOptions, ThemeContextProvider } from '../../design';
import Table from './table';
import { useNavigableRows } from './useNavigableRows';
import { usePerconaTableUrlState } from './usePerconaTableUrlState';

type TestRow = {
  id: string;
  name: string;
  group: string;
};

const TEST_DATA: TestRow[] = Array.from({ length: 12 }, (_, i) => ({
  id: String(i + 1).padStart(2, '0'),
  name: `item-${String(i + 1).padStart(2, '0')}`,
  group: i === 0 || i === 11 ? 'edge' : i % 2 === 0 ? 'even' : 'odd',
}));

const columns: MRT_ColumnDef<TestRow>[] = [
  { header: 'Name', accessorKey: 'name' },
  {
    header: 'Group',
    accessorKey: 'group',
    filterVariant: 'select',
    filterSelectOptions: ['edge', 'even', 'odd'],
  },
];

const getNavigableIds = () => screen.getByTestId('navigable-ids').textContent ?? '';

describe('usePerconaTableUrlState + useNavigableRows', () => {
  it('composes without duplicate table handlers', async () => {
    const ComposedHarness = () => {
      const [searchParams, setSearchParams] = useState(() => new URLSearchParams());
      const { tableState, tableProps: urlTableProps } = usePerconaTableUrlState({
        searchParams,
        setSearchParams,
        debounceMs: 0,
      });
      const { navigableRows, tableProps: navTableProps } = useNavigableRows({
        data: TEST_DATA,
        tableState,
      });

      return (
        <>
          <Table
            {...urlTableProps}
            {...navTableProps}
            tableName="navigable-rows-composed-test"
            columns={columns}
            data={TEST_DATA}
            enableColumnFilters
            initialState={{ showColumnFilters: true }}
          />
          <button
            type="button"
            data-testid="apply-filter"
            onClick={() => urlTableProps.onColumnFiltersChange([{ id: 'group', value: 'edge' }])}
          >
            Filter
          </button>
          <div data-testid="navigable-ids">{navigableRows.map((row) => row.id).join(',')}</div>
          <div data-testid="has-handlers">
            {String(
              !!navTableProps.onColumnFiltersChange &&
                !!navTableProps.onGlobalFilterChange &&
                !!navTableProps.onSortingChange
            )}
          </div>
          <div data-testid="search">{searchParams.toString()}</div>
        </>
      );
    };

    render(
      <ThemeContextProvider themeOptions={baseThemeOptions}>
        <ComposedHarness />
      </ThemeContextProvider>
    );

    expect(screen.getByTestId('has-handlers').textContent).toBe('false');

    await act(async () => {
      screen.getByTestId('apply-filter').click();
    });

    await waitFor(() => {
      expect(getNavigableIds()).toBe('01,12');
      expect(screen.getByTestId('search').textContent).toContain('f.group=edge');
    });
  });
});
