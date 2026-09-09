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
 * `TaskHistoryResponse` carries no error field today — status, started_at,
 * finished_at, duration, has_logs and log_capture, and nothing else. SEP-2000
 * adds one; until it lands every candidate here is absent and
 * {@link runFailureReason} returns null, which the UI renders as a pointer to
 * the log rather than as an empty error.
 *
 * Several names are accepted rather than one so the frontend does not have to
 * ship in lockstep with whichever name the side-car settles on. Order is
 * preference: the most specific name wins if two are ever populated.
 *
 * TODO(SEP-2000): once the side-car has shipped a field, narrow this to that
 * one name and drop the tracking fallback. This file is a bet on a shape
 * nobody has committed to yet, and it lives in vendored code — leaving three
 * guesses in place after the real name is known is how it ends up silently
 * matching the wrong thing through a future sync.
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
