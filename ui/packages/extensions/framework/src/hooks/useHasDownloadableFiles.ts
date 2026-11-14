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

import {
  useTaskHistoryFiles,
  type UseTaskHistoryFilesOptions,
} from './useTaskHistoryFiles';

/**
 * Whether a run has user-visible files worth offering a download for.
 *
 * ``has_logs`` is the wrong signal: logs can exist when the output directory is
 * empty (e.g. only the hidden ``.pmm-extensions-run-result.json`` marker). Probes the
 * files API and reports `true` only once a non-empty listing is confirmed, so
 * a caller can hide its Files action rather than open to an empty dialog.
 *
 * `canProbe` is a separate input rather than folded into `taskHistoryId` being
 * `null`: `TaskHistoryTable` gates on the run being finished and declaring an
 * output path, and ATW's results pane has no equivalent of the latter — each
 * caller decides when probing is worth the request.
 */
export function useHasDownloadableFiles(
  taskHistoryId: number | null | undefined,
  canProbe: boolean,
  options?: UseTaskHistoryFilesOptions
): boolean {
  const { data, isLoading, isError } = useTaskHistoryFiles(
    canProbe ? taskHistoryId : null,
    options
  );
  return (
    canProbe &&
    !isLoading &&
    !isError &&
    Boolean(data) &&
    Object.keys(data ?? {}).length > 0
  );
}
