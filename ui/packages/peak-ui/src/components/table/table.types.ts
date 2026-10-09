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

export interface TableProps<T extends MRT_RowData> extends MRT_TableOptions<T> {
  /** Gives the parent access to the table instance, for example to read the visible rows or reset filters. */
  tableInstanceRef?: MutableRefObject<MRT_TableInstance<T> | null>;
  /** Message shown in the body when there are no rows at all. Defaults to "No data". */
  noDataMessage?: string;
  /** Message shown in the body when search or filters do not reveal matching rows. Defaults to "No data found". */
  emptyFilterResultsMessage?: string;
  /** Hides the "expand all" control in the header of expandable tables, so rows expand one by one. */
  hideExpandAllIcon?: boolean;
  /** Unique name of this table; the user's column visibility choices are saved under it, so renaming it forgets them. */
  tableName: string;
  /** Replaces `noDataMessage` when there are no rows at all, for example an illustration with an explanatory text and a call to action. */
  emptyState?: React.ReactNode;
  /** MUI Alert props for the no-data and no-results messages, for example to change the severity. */
  noDataAlertProps?: AlertProps;
  /** Makes whole rows clickable: shows a pointer cursor and calls `rowHoverAction` on click. */
  enableRowHoverAction?: boolean;
  /** Called with the clicked row when `enableRowHoverAction` is on, for example to open a details pane. */
  rowHoverAction?: (row: MRT_Row<T>) => void;
}
