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
  EventStreamContentType,
  fetchEventSource,
} from '@microsoft/fetch-event-source';
import {
  apiClient,
  emitUnauthorized,
  getToken,
  refreshAccessToken,
} from '@sep/api';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useMemo, useRef, useState } from 'react';

export interface ExecutionEvent {
  timestamp: string;
  type: string;
  description: string;
  step?: string | null;
}

export interface ExecutionEventsState {
  events: ExecutionEvent[];
  eventsByStep: Record<string, ExecutionEvent[]>;
  stepOrder: string[];
  isLoading: boolean;
  error?: unknown;
}

const STEPLESS_KEY = '';

/** Consecutive transient stream failures tolerated before giving up. */
const MAX_TRANSIENT_STREAM_FAILURES = 5;

// See useTaskLogs.ts for the rationale behind these local sentinel classes.
class StreamRetriableAfterRefresh extends Error {}
class StreamFatalError extends Error {}

function compositeKey(ev: ExecutionEvent): string {
  return `${ev.timestamp ?? ''}|${ev.type ?? ''}|${ev.description ?? ''}|${ev.step ?? ''}`;
}

function groupByStep(events: ExecutionEvent[]): {
  eventsByStep: Record<string, ExecutionEvent[]>;
  stepOrder: string[];
} {
  const eventsByStep: Record<string, ExecutionEvent[]> = {};
  const stepOrder: string[] = [];
  for (const ev of events) {
    const step = ev.step ?? '';
    const key = step !== '' ? String(step) : STEPLESS_KEY;
    if (!eventsByStep[key]) {
      eventsByStep[key] = [];
      stepOrder.push(key);
    }
    eventsByStep[key].push(ev);
  }
  return { eventsByStep, stepOrder };
}

async function fetchExecutionEvents(
  taskHistoryId: string
): Promise<ExecutionEvent[]> {
  // Use apiClient (Bearer interceptor + 401-refresh) with baseURL: '' to reach
  // the legacy /execution-events/{id} route that is not under /api.
  const res = await apiClient.get<ExecutionEvent[]>(
    `/execution-events/${taskHistoryId}`,
    {
      baseURL: '',
    }
  );
  return Array.isArray(res.data) ? res.data : [];
}

/**
 * Execution events for a task history.
 *
 * Running tasks stream via SSE /stream-logs/{id}/execution-events.
 * Completed tasks fetch REST /execution-events/{id} through react-query for
 * consistent error/loading/cache semantics with the rest of @sep/framework.
 *
 * Both paths attach the Bearer token from @sep/api's token provider so the
 * SPA OAuth session is honoured without a session cookie.
 */
export function useExecutionEvents(
  taskHistoryId: number | string | undefined,
  isRunning: boolean
): ExecutionEventsState {
  const idStr =
    taskHistoryId === undefined ||
    taskHistoryId === null ||
    taskHistoryId === ''
      ? undefined
      : encodeURIComponent(String(taskHistoryId));

  // ── Completed tasks: REST via react-query ─────────────────────────────
  const query = useQuery<ExecutionEvent[]>({
    queryKey: ['execution-events', idStr],
    queryFn: () => fetchExecutionEvents(idStr as string),
    enabled: !isRunning && idStr !== undefined,
  });

  // ── Running tasks: SSE ────────────────────────────────────────────────
  const [sseEvents, setSseEvents] = useState<ExecutionEvent[]>([]);
  const [sseError, setSseError] = useState<unknown>(undefined);
  const [sseLoading, setSseLoading] = useState(false);

  const seenKeysRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    if (!isRunning || idStr === undefined) {
      return;
    }

    seenKeysRef.current = new Set();
    setSseEvents([]);
    setSseError(undefined);
    setSseLoading(true);

    const ctrl = new AbortController();
    let currentToken = getToken() ?? '';
    let refreshAttempted = false;
    // Guards state setters so stale callbacks from an aborted stream cannot
    // write into a subsequent stream's state (e.g. on rapid id changes).
    let disposed = false;
    // Tracks intentional terminal events (finish / sep-error) so onclose can
    // distinguish a clean shutdown from an unexpected connection drop.
    let terminatedCleanly = false;
    // Consecutive transient failures, reset on every successful open. Bounds the
    // otherwise-infinite fetchEventSource retry loop.
    let transientFailures = 0;

    fetchEventSource(`/stream-logs/${idStr}/execution-events`, {
      signal: ctrl.signal,
      openWhenHidden: true,

      fetch: (input, init) => {
        const headers = new Headers(init?.headers as HeadersInit | undefined);
        if (currentToken) {
          headers.set('Authorization', `Bearer ${currentToken}`);
        }
        return globalThis.fetch(input as RequestInfo, { ...init, headers });
      },

      onopen: async (response) => {
        const contentType = response.headers.get('content-type') ?? '';
        if (response.ok && contentType.includes(EventStreamContentType)) {
          refreshAttempted = false;
          transientFailures = 0;
          return;
        }
        let isAuthFailure = false;
        if (response.ok) {
          isAuthFailure = true;
        } else if (response.status === 401 && !refreshAttempted) {
          refreshAttempted = true;
          const newToken = await refreshAccessToken();
          if (newToken) {
            currentToken = newToken;
            throw new StreamRetriableAfterRefresh();
          }
          isAuthFailure = true;
        } else if (response.status === 401) {
          isAuthFailure = true;
        }
        if (isAuthFailure) {
          emitUnauthorized();
        }
        const message = response.ok
          ? `Stream endpoint returned non-SSE content: ${contentType || 'unknown'}`
          : `Stream open failed with status ${response.status}`;
        // Terminal open failure — set error, abort, throw fatal sentinel so
        // onerror re-throws and permanently stops the retry loop.
        if (!disposed) {
          setSseError({ message });
          setSseLoading(false);
        }
        terminatedCleanly = true;
        ctrl.abort();
        throw new StreamFatalError(message);
      },

      onmessage: (ev) => {
        if (disposed || ev.data === '') {
          return;
        }

        if (ev.event === 'finish') {
          terminatedCleanly = true;
          setSseLoading(false);
          ctrl.abort();
          return;
        }

        if (ev.event === 'sep-error') {
          terminatedCleanly = true;
          let payload: unknown = ev.data;
          try {
            payload = JSON.parse(ev.data);
          } catch {
            // keep raw
          }
          setSseError(payload);
          setSseLoading(false);
          ctrl.abort();
          return;
        }

        // Default message — execution event
        let payload: ExecutionEvent;
        try {
          payload = JSON.parse(ev.data) as ExecutionEvent;
        } catch {
          return;
        }
        const key = compositeKey(payload);
        if (seenKeysRef.current.has(key)) {
          return;
        }
        seenKeysRef.current.add(key);
        setSseEvents((prev) => [...prev, payload]);
        setSseLoading(false);
      },

      onerror: (err): number | undefined => {
        if (err instanceof StreamRetriableAfterRefresh) {
          return 0;
        }
        if (err instanceof StreamFatalError) {
          // State already set in onopen; re-throw to stop the retry loop.
          throw err;
        }
        // Transient network error — let the library retry with backoff silently
        // while the failure count stays under the cap. Do not set sseError: the
        // previous events remain visible and the next successful open resumes
        // streaming.
        transientFailures += 1;
        if (transientFailures < MAX_TRANSIENT_STREAM_FAILURES) {
          return undefined;
        }
        // Cap reached: a persistently failing endpoint (500, DNS failure) would
        // otherwise reconnect forever while the panel sits in its loading state,
        // because nothing here clears it. Surface the failure and re-throw to
        // stop the retry loop.
        if (!disposed) {
          setSseError({
            message: `Execution events stream failed after ${MAX_TRANSIENT_STREAM_FAILURES} attempts.`,
          });
          setSseLoading(false);
        }
        terminatedCleanly = true;
        ctrl.abort();
        throw err;
      },

      onclose: () => {
        if (disposed || terminatedCleanly) {
          return;
        }
        // Stream closed unexpectedly without a finish/sep-error frame.
        setSseError({ message: 'Execution events stream connection closed.' });
        setSseLoading(false);
      },
    }).catch(() => {
      // Expected abort on unmount / isRunning flip. StreamFatalError re-thrown
      // from onerror also lands here; state is already set.
    });

    return () => {
      disposed = true;
      ctrl.abort();
    };
  }, [idStr, isRunning]);

  const events = isRunning ? sseEvents : (query.data ?? []);
  const error = isRunning ? sseError : (query.error ?? undefined);
  const isLoading = isRunning ? sseLoading : query.isLoading;

  const { eventsByStep, stepOrder } = useMemo(
    () => groupByStep(events),
    [events]
  );

  return { events, eventsByStep, stepOrder, isLoading, error };
}
