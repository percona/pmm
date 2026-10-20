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

import { vi } from 'vitest';

const encoder = new TextEncoder();

/**
 * Per-request handle that lets tests push SSE frames into the stream and
 * inspect what headers the hook sent.
 */
export interface SseStreamHandle {
  readonly url: string;
  /** Raw request headers (including Authorization) */
  readonly requestHeaders: Record<string, string>;
  /** Signal from the internal AbortController that fetchEventSource created */
  readonly signal: AbortSignal | undefined;
  pushMessage(data: unknown): void;
  pushNamed(event: string, data: unknown): void;
  pushRaw(frame: string): void;
  close(): void;
  errorStream(reason?: unknown): void;
}

interface QueuedResponse {
  status: number;
  contentType?: string;
}

/**
 * Create a mock `fetch` that returns controllable SSE streams.
 *
 * Call `install()` to stub `globalThis.fetch`. For each network request the
 * hook makes, a `SseStreamHandle` is pushed into `pending` so tests can drive
 * the stream with `pushMessage` / `pushNamed` / `close`.
 *
 * Non-200 responses (e.g. 401) can be pre-queued via `queueResponse` — the
 * next fetch call dequeues that config instead of creating a stream.
 */
export function mockStreamFetch() {
  const pending: SseStreamHandle[] = [];
  const responseQueue: QueuedResponse[] = [];

  const fetchSpy = vi.fn(
    (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
      const url =
        typeof input === 'string'
          ? input
          : input instanceof URL
            ? input.href
            : (input as Request).url;

      const cfg = responseQueue.shift();

      if (cfg && cfg.status !== 200) {
        // Non-streaming response (e.g. 401 for auth tests)
        return Promise.resolve(
          new Response(null, {
            status: cfg.status,
            headers: { 'Content-Type': cfg.contentType ?? 'application/json' },
          })
        );
      }

      // Normal SSE response with a controllable ReadableStream
      let ctrl!: ReadableStreamDefaultController<Uint8Array>;
      const body = new ReadableStream<Uint8Array>({
        start(c) {
          ctrl = c;
        },
      });

      const rawHeaders = init?.headers;
      let requestHeaders: Record<string, string> = {};
      if (rawHeaders instanceof Headers) {
        requestHeaders = Object.fromEntries(rawHeaders.entries());
      } else if (Array.isArray(rawHeaders)) {
        requestHeaders = Object.fromEntries(rawHeaders);
      } else if (rawHeaders) {
        requestHeaders = rawHeaders as Record<string, string>;
      }

      const handle: SseStreamHandle = {
        url,
        requestHeaders,
        signal: init?.signal ?? undefined,
        pushMessage(data) {
          ctrl.enqueue(encoder.encode(`data: ${JSON.stringify(data)}\n\n`));
        },
        pushNamed(event, data) {
          ctrl.enqueue(
            encoder.encode(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`)
          );
        },
        pushRaw(frame) {
          ctrl.enqueue(encoder.encode(frame));
        },
        close() {
          ctrl.close();
        },
        errorStream(reason) {
          ctrl.error(reason);
        },
      };

      pending.push(handle);

      return Promise.resolve(
        new Response(body, {
          status: 200,
          headers: { 'Content-Type': 'text/event-stream' },
        })
      );
    }
  );

  return {
    /** One entry per fetch call, in order. */
    pending,
    /** The raw vitest spy — use for call count assertions etc. */
    fetchSpy,
    /** Replace `globalThis.fetch` with this mock. */
    install() {
      vi.stubGlobal('fetch', fetchSpy);
    },
    /** Queue a non-200 response for the *next* fetch call. */
    queueResponse(opts: QueuedResponse) {
      responseQueue.push(opts);
    },
  };
}

/**
 * Drain the microtask queue by chaining several `queueMicrotask` calls.
 *
 * Uses `queueMicrotask` (not `setTimeout`) so it works correctly even when
 * `vi.useFakeTimers` is active (fake timers intercept `setTimeout` but not
 * the microtask queue). Four levels cover the nested `await` chain inside
 * `fetchEventSource` (fetch → onopen → async callbacks → onerror).
 */
export function flushPromises(): Promise<void> {
  return new Promise((resolve) => {
    queueMicrotask(() =>
      queueMicrotask(() => queueMicrotask(() => queueMicrotask(resolve)))
    );
  });
}
