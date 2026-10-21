// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import { AlertProps } from '@mui/material';
import {
  type MRT_Row,
  type MRT_RowData,
  type MRT_TableInstance,
  type MRT_TableOptions,
} from 'material-react-table';
import { type MutableRefObject } from 'react';

export type TableNoDataVariant = 'empty-state' | 'alert';

export interface TableProps<T extends MRT_RowData> extends MRT_TableOptions<T> {
  tableInstanceRef?: MutableRefObject<MRT_TableInstance<T> | null>;
  /** Title of the empty state shown when `data` is empty. */
  noDataMessage?: string;
  emptyFilterResultsMessage?: string;
  hideExpandAllIcon?: boolean;
  tableName: string;
  /** Replaces the default empty state; pass an `EmptyState` with a description, an illustration or actions. */
  emptyState?: React.ReactNode;
  /**
   * How an empty table reads: `empty-state` renders a small `EmptyState`; `alert` keeps the
   * info alert from before `EmptyState` existed, for tables not yet moved to it.
   * @default 'empty-state'
   */
  noDataVariant?: TableNoDataVariant;
  /** Props for the info alert: the empty-filter-results one, and the no-data one under `noDataVariant="alert"`. */
  noDataAlertProps?: AlertProps;
  enableRowHoverAction?: boolean;
  rowHoverAction?: (row: MRT_Row<T>) => void;
}
