import type {
  MRT_ColumnFiltersState,
  MRT_Updater,
  MRT_VisibilityState,
} from '@percona/peak-ui';
import { useLocalStorage } from 'hooks/utils/useLocalStorage';

interface TableFilters {
  state: { columnFilters: MRT_ColumnFiltersState };
  onColumnFiltersChange: (updater: MRT_Updater<MRT_ColumnFiltersState>) => void;
}

// Owns the table's column visibility so an ID column can start hidden: it is
// shown from the columns menu, or while filtered on (e.g. from a link on the
// other tab) so the filter stays visible.
export const useIdColumnVisibility = (
  tableName: string,
  idColumn: string,
  { state: { columnFilters }, onColumnFiltersChange }: TableFilters
) => {
  const [visibility, setVisibility] = useLocalStorage<MRT_VisibilityState>(
    `${tableName}.columns`,
    { [idColumn]: false }
  );
  const isFiltered = columnFilters.some(({ id }) => id === idColumn);
  const columnVisibility = isFiltered
    ? { ...visibility, [idColumn]: true }
    : visibility;

  const onColumnVisibilityChange = (
    updater: MRT_Updater<MRT_VisibilityState>
  ) => {
    const next =
      typeof updater === 'function' ? updater(columnVisibility) : updater;

    // A hidden column could not show the filter narrowing the rows
    if (isFiltered && next[idColumn] === false) {
      onColumnFiltersChange((filters) =>
        filters.filter(({ id }) => id !== idColumn)
      );
    }

    // While filtered, a visible ID column is only forced and not the user's choice
    setVisibility(
      isFiltered && next[idColumn]
        ? { ...next, [idColumn]: visibility[idColumn] ?? false }
        : next
    );
  };

  return { columnVisibility, onColumnVisibilityChange };
};
