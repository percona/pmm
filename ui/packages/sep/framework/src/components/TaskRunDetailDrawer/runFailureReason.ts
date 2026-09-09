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

import type { TaskHistoryEntry } from '../../hooks/useTaskHistory';

/**
 * Field names a failure reason may arrive under.
 *
 * The bet this file was written as has now settled: `TaskHistoryResponse`
 * carries `failure_reason`, and the vendored spec has been synced to a
 * side-car that serves it. An older side-car still sends none of these, and
 * {@link runFailureReason} returns null, which the UI renders as a pointer to
 * the log rather than as an empty error.
 *
 * Order is preference, and it is now load-bearing in a way it was not when
 * every name was hypothetical: `error_message` sorts ahead of the real field,
 * so a side-car populating both would be read on the wrong one.
 *
 * TODO: narrow this to `failure_reason` and drop the tracking fallback. The
 * condition the original note put this off until — the side-car shipping a
 * field — is met; what remains is rewriting the cases in
 * `runFailureReason.test.ts` that are built around the multi-key and
 * tracking-bag behaviour, which belongs with the drawer rather than with a
 * sync.
 */
const FAILURE_REASON_KEYS = [
  'error_message',
  'failure_reason',
  'error',
] as const;

function firstNonEmptyString(source: Record<string, unknown>): string | null {
  for (const key of FAILURE_REASON_KEYS) {
    const value = source[key];
    if (typeof value === 'string' && value.trim() !== '') {
      return value.trim();
    }
  }

  return null;
}

/**
 * The reason a run failed, or null when the wire does not carry one.
 *
 * Reads the history row first, then `execution_request.tracking` — an untyped
 * escape hatch the backend already uses for per-run bookkeeping, and the likely
 * home for a reason that arrives before the response schema gains a field.
 */
export function runFailureReason(
  entry: TaskHistoryEntry | null | undefined
): string | null {
  if (!entry) {
    return null;
  }

  const direct = firstNonEmptyString(
    entry as unknown as Record<string, unknown>
  );
  if (direct !== null) {
    return direct;
  }

  const tracking = entry.execution_request?.tracking;
  if (tracking !== null && typeof tracking === 'object') {
    return firstNonEmptyString(tracking as Record<string, unknown>);
  }

  return null;
}

/**
 * The first line of a failure reason, for surfaces that show a summary.
 *
 * A reason is often a multi-line traceback or a wall of stderr. The Overview
 * block shows one line and sends the reader to the log for the rest, so the
 * card keeps its height whatever the backend sends.
 */
export function firstLine(reason: string | null): string | null {
  if (reason === null) {
    return null;
  }

  const [line] = reason.split('\n');

  return line?.trim() ? line.trim() : null;
}
