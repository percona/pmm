import { ReactElement } from 'react';

export interface TableProps<T> {
  rows: T[];
  rowId: keyof T;
  columns: Column<T>[];
  emptyMessage?: string;
  isLoading?: boolean;
}

export interface Column<T> {
  field: keyof T;
  name: string;
  cell?: (item: T) => ReactElement;
}
