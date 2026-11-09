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

import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { SEARCH_DEBOUNCE_MS, useDebouncedValue } from './useDebouncedValue';

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

/** Advance timers inside `act` so the resulting state update is flushed. */
function advance(ms: number) {
  act(() => {
    vi.advanceTimersByTime(ms);
  });
}

describe('useDebouncedValue', () => {
  it('publishes the initial value immediately', () => {
    const { result } = renderHook(() => useDebouncedValue('seed'));

    expect(result.current).toBe('seed');
  });

  it('withholds a new value until the delay elapses', () => {
    const { result, rerender } = renderHook(
      ({ value }) => useDebouncedValue(value),
      {
        initialProps: { value: '' },
      }
    );

    rerender({ value: 'pg' });
    expect(result.current).toBe('');

    advance(SEARCH_DEBOUNCE_MS - 1);
    expect(result.current).toBe('');

    advance(1);
    expect(result.current).toBe('pg');
  });

  it('publishes once per pause, not once per change', () => {
    const { result, rerender } = renderHook(
      ({ value }) => useDebouncedValue(value),
      {
        initialProps: { value: '' },
      }
    );

    for (const value of ['p', 'pg', 'pgs', 'pgst']) {
      rerender({ value });
      advance(SEARCH_DEBOUNCE_MS - 50);
    }
    // Every keystroke landed inside the window, so nothing has been published.
    expect(result.current).toBe('');

    advance(50);
    expect(result.current).toBe('pgst');
  });

  it('honours a caller-supplied delay', () => {
    const { result, rerender } = renderHook(
      ({ value }) => useDebouncedValue(value, 1000),
      {
        initialProps: { value: '' },
      }
    );

    rerender({ value: 'slow' });
    advance(SEARCH_DEBOUNCE_MS);
    expect(result.current).toBe('');

    advance(1000 - SEARCH_DEBOUNCE_MS);
    expect(result.current).toBe('slow');
  });

  it('restarts the window when the delay itself changes', () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebouncedValue(value, delay),
      {
        initialProps: { value: 'a', delay: SEARCH_DEBOUNCE_MS },
      }
    );

    rerender({ value: 'b', delay: SEARCH_DEBOUNCE_MS });
    advance(SEARCH_DEBOUNCE_MS - 100);

    rerender({ value: 'b', delay: 1000 });
    advance(100);
    expect(result.current).toBe('a');

    advance(900);
    expect(result.current).toBe('b');
  });

  it('clears the pending timer on unmount', () => {
    const clearTimeoutSpy = vi.spyOn(globalThis, 'clearTimeout');
    const { rerender, unmount } = renderHook(
      ({ value }) => useDebouncedValue(value),
      {
        initialProps: { value: '' },
      }
    );

    rerender({ value: 'gone' });
    clearTimeoutSpy.mockClear();
    unmount();

    expect(clearTimeoutSpy).toHaveBeenCalledTimes(1);

    // The timer never fires, so no state update is attempted after unmount.
    expect(() => advance(SEARCH_DEBOUNCE_MS)).not.toThrow();
    clearTimeoutSpy.mockRestore();
  });
});
