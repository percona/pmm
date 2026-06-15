import { act, render, screen, waitFor } from '@testing-library/react';
import { type MRT_ColumnDef } from 'material-react-table';
import { describe, expect, it, vi } from 'vitest';
import { baseThemeOptions, ThemeContextProvider } from '../../design';
import Table from './table';
import { useNavigableRows, type NavigableRowsScope } from './useNavigableRows';

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

function NavigableTableHarness({
  scope,
  onChange,
}: {
  scope?: NavigableRowsScope;
  onChange?: (rows: TestRow[]) => void;
}) {
  const { navigableRows, tableProps } = useNavigableRows({
    data: TEST_DATA,
    scope,
    onChange,
  });

  return (
    <>
      <Table
        {...tableProps}
        tableName="navigable-rows-test"
        columns={columns}
        data={TEST_DATA}
        enableColumnFilters
        initialState={{ showColumnFilters: true }}
      />
      <div data-testid="navigable-ids">{navigableRows.map((row) => row.id).join(',')}</div>
    </>
  );
}

const renderHarness = (props?: { scope?: NavigableRowsScope; onChange?: (rows: TestRow[]) => void }) =>
  render(
    <ThemeContextProvider themeOptions={baseThemeOptions}>
      <NavigableTableHarness {...props} />
    </ThemeContextProvider>
  );

describe('useNavigableRows', () => {
  it('returns all filtered rows across pages by default', async () => {
    renderHarness();

    await waitFor(() => {
      expect(getNavigableIds()).toBe('01,02,03,04,05,06,07,08,09,10,11,12');
    });
  });

  it('returns only current page rows when scope is currentPage', async () => {
    renderHarness({ scope: 'currentPage' });

    await waitFor(() => {
      expect(getNavigableIds()).toBe('01,02,03,04,05,06,07,08,09,10');
    });
  });

  it('keeps filtered rows across pages', async () => {
    const FilterHarness = () => {
      const { navigableRows, tableProps } = useNavigableRows({ data: TEST_DATA });

      return (
        <>
          <Table
            {...tableProps}
            tableName="navigable-rows-filter-test"
            columns={columns}
            data={TEST_DATA}
            enableColumnFilters
            initialState={{ showColumnFilters: true }}
          />
          <button
            type="button"
            data-testid="apply-filter"
            onClick={() =>
              tableProps.onColumnFiltersChange([{ id: 'group', value: 'edge' }])
            }
          >
            Filter
          </button>
          <div data-testid="navigable-ids">{navigableRows.map((row) => row.id).join(',')}</div>
        </>
      );
    };

    render(
      <ThemeContextProvider themeOptions={baseThemeOptions}>
        <FilterHarness />
      </ThemeContextProvider>
    );

    await act(async () => {
      screen.getByTestId('apply-filter').click();
    });

    await waitFor(() => {
      expect(getNavigableIds()).toBe('01,12');
    });
  });

  it('reflects sort order in navigable rows', async () => {
    const SortHarness = () => {
      const { navigableRows, tableProps } = useNavigableRows({ data: TEST_DATA });

      return (
        <>
          <Table
            {...tableProps}
            tableName="navigable-rows-sort-test"
            columns={columns}
            data={TEST_DATA}
          />
          <button
            type="button"
            data-testid="apply-sort"
            onClick={() => tableProps.onSortingChange([{ id: 'name', desc: true }])}
          >
            Sort
          </button>
          <div data-testid="navigable-ids">{navigableRows.map((row) => row.id).join(',')}</div>
        </>
      );
    };

    render(
      <ThemeContextProvider themeOptions={baseThemeOptions}>
        <SortHarness />
      </ThemeContextProvider>
    );

    await act(async () => {
      screen.getByTestId('apply-sort').click();
    });

    await waitFor(() => {
      expect(getNavigableIds()).toBe('12,11,10,09,08,07,06,05,04,03,02,01');
    });
  });

  it('calls onChange when filter changes', async () => {
    const onChange = vi.fn();
    const FilterHarness = () => {
      const { navigableRows, tableProps } = useNavigableRows({
        data: TEST_DATA,
        onChange,
      });

      return (
        <>
          <Table
            {...tableProps}
            tableName="navigable-rows-onchange-test"
            columns={columns}
            data={TEST_DATA}
            enableColumnFilters
          />
          <button
            type="button"
            data-testid="apply-filter"
            onClick={() =>
              tableProps.onColumnFiltersChange([{ id: 'group', value: 'edge' }])
            }
          >
            Filter
          </button>
          <div data-testid="navigable-ids">{navigableRows.map((row) => row.id).join(',')}</div>
        </>
      );
    };

    render(
      <ThemeContextProvider themeOptions={baseThemeOptions}>
        <FilterHarness />
      </ThemeContextProvider>
    );

    await waitFor(() => {
      expect(onChange).toHaveBeenCalled();
    });

    onChange.mockClear();

    await act(async () => {
      screen.getByTestId('apply-filter').click();
    });

    await waitFor(() => {
      expect(getNavigableIds()).toBe('01,12');
      expect(onChange).toHaveBeenCalled();
      const lastCall = onChange.mock.calls.at(-1)?.[0] as TestRow[];
      expect(lastCall.map((row) => row.id)).toEqual(['01', '12']);
    });
  });
});
