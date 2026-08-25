/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import type {
  PaginatedTaskHistory,
  TaskHistoryEntry,
  TaskHistoryStatus,
} from '../../hooks/useTaskHistory';

export type { PaginatedTaskHistory, TaskHistoryEntry, TaskHistoryStatus };

interface TaskHistoryTableBaseProps {
  /** Optional task name to scope the listing to a single task. */
  taskName?: string;
  /** Optional explicit data — bypasses the internal React Query hook (used in stories/tests). */
  data?: TaskHistoryEntry[];
  /** Server-side status filter (applied via the React Query hook). */
  statusFilter?: TaskHistoryStatus | null;
  /** Loading flag (only honored when `data` is provided). */
  isLoading?: boolean;
  /** Override polling interval in ms. Default 5000. */
  pollingIntervalMs?: number;
  /** Force-disable polling regardless of running tasks. */
  disablePolling?: boolean;
  /** Resolver from Casdoor user id → display name. */
  resolveUserName?: (userId: string | null | undefined) => string;
  /** Action callback: view logs for a row. */
  onViewLogs?: (entry: TaskHistoryEntry) => void;
  /**
   * Whether a stop request is currently in flight (presentational mode only).
   * Drives the Stop button's spinner/disabled state; defaults to `false`. The
   * connected variant derives this from its internal mutation instead.
   */
  isStopping?: boolean;
  /** Action callback: open download dialog. */
  onDownloadFiles?: (entry: TaskHistoryEntry) => void;
  /** Action callback: navigate to chained task by name. */
  onChainItemClick?: (
    taskName: string,
    index: number,
    entry: TaskHistoryEntry
  ) => void;
  /** Hide the Task Name column (useful when scoped to a single task). */
  hideTaskNameColumn?: boolean;
}

/**
 * Who owns the stop mutation, and therefore who owns reporting its failure.
 *
 * A caller that supplies `onStopTask` owns the mutation, so it must also supply
 * `actionError`: the stop confirmation closes as soon as it is confirmed, the
 * request settles with the user looking at the table, and this is the only
 * place the refusal can appear. Making it a union rather than two optional
 * props means dropping the error is a type error rather than a silent failure.
 *
 * Omitting `onStopTask` selects the connected variant's internal mutation,
 * which reports for itself and would ignore these props — so the union forbids
 * them there rather than accepting a value that goes nowhere.
 */
export type TaskHistoryStopContract =
  | {
      onStopTask?: never;
      actionError?: never;
      onDismissActionError?: never;
    }
  | {
      /** Action callback: stop a running task. */
      onStopTask: (entry: TaskHistoryEntry) => void;
      /**
       * Failure of an action fired from this table, rendered as an alert above
       * the rows. Pass the owning mutation's `error`; `null` when there is
       * none.
       */
      actionError: unknown;
      /** Dismiss handler for `actionError` (e.g. the mutation's `reset`). */
      onDismissActionError?: () => void;
    };

export type TaskHistoryTableProps = TaskHistoryTableBaseProps &
  TaskHistoryStopContract;

export type { TaskHistoryTableBaseProps };
