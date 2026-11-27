import type { MRT_RowData, MRT_TableOptions } from 'material-react-table';

export const DEFAULT_TABLE_OPTIONS = {
  enableStickyHeader: true,
  // Sorting, filters and columns stay reachable from the headers and toolbar,
  // while the header space goes to the column names
  enableColumnActions: false,
  positionActionsColumn: 'last',
  positionToolbarAlertBanner: 'none',
  globalFilterFn: 'contains',
} satisfies Partial<MRT_TableOptions<MRT_RowData>>;
