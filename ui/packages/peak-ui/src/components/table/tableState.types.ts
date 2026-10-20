import {
  MRT_ColumnFiltersState,
  MRT_PaginationState,
  MRT_SortingState,
  MRT_Updater,
} from 'material-react-table';

export interface TableStateValues {
  columnFilters: MRT_ColumnFiltersState;
  globalFilter: string;
  sorting: MRT_SortingState;
  pagination: MRT_PaginationState;
}

export interface TableControlledState {
  state: TableStateValues;
  onColumnFiltersChange: (updater: MRT_Updater<MRT_ColumnFiltersState>) => void;
  onGlobalFilterChange: (updater: MRT_Updater<string>) => void;
  onSortingChange: (updater: MRT_Updater<MRT_SortingState>) => void;
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
