import {
  MRT_ColumnFiltersState,
  MRT_PaginationState,
  MRT_SortingState,
  MRT_Updater,
} from 'material-react-table';

export interface TableStateValues {
  /** Active per-column filters. */
  columnFilters: MRT_ColumnFiltersState;
  /** Text typed in the search box. */
  globalFilter: string;
  /** Active sort order, one entry per sorted column. */
  sorting: MRT_SortingState;
  /** Current page and rows per page. */
  pagination: MRT_PaginationState;
}

export interface TableControlledState {
  /** Current filters, search text, sorting, and pagination, passed to `Table` as its `state`. */
  state: TableStateValues;
  /** Receives the new per-column filters when the user changes them. */
  onColumnFiltersChange: (updater: MRT_Updater<MRT_ColumnFiltersState>) => void;
  /** Receives the new search text when the user types in the search box. */
  onGlobalFilterChange: (updater: MRT_Updater<string>) => void;
  /** Receives the new sort order when the user sorts a column. */
  onSortingChange: (updater: MRT_Updater<MRT_SortingState>) => void;
  /** Receives the new page or rows per page when the user paginates. */
  onPaginationChange: (updater: MRT_Updater<MRT_PaginationState>) => void;
}

export type NavigableTableState = Pick<
  TableControlledState,
  'state' | 'onColumnFiltersChange' | 'onGlobalFilterChange' | 'onSortingChange'
>;

export const DEFAULT_TABLE_STATE: TableStateValues = {
  columnFilters: [],
  globalFilter: '',
  sorting: [],
  pagination: { pageIndex: 0, pageSize: 10 },
};
