import { type TableProps } from './table.types';

export { default } from './table';
export { TableProps };

export {
  default as useNavigableRows,
  type NavigableRowsScope,
  type UseNavigableRowsOptions,
  type UseNavigableRowsResult,
  type UseNavigableRowsTableProps,
} from './useNavigableRows';

export {
  default as useDetailsPaneNavigation,
  type UseDetailsPaneNavigationOptions,
  type UseDetailsPaneNavigationResult,
} from './useDetailsPaneNavigation';

// Re-export common Material React Table types so consumers don't have to
// reach into the transitive dependency directly. Keep this list
// conservative and additive — adding new symbols is safe, removing is not.
export type {
  MRT_Cell,
  MRT_ColumnDef,
  MRT_ColumnFiltersState,
  MRT_PaginationState,
  MRT_Row,
  MRT_RowSelectionState,
  MRT_SortingState,
  MRT_TableInstance,
  MRT_Updater,
  MRT_VisibilityState,
  MaterialReactTableProps,
} from 'material-react-table';
