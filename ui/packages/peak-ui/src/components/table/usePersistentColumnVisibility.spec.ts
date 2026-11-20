import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import usePersistentColumnVisibility from './usePersistentColumnVisibility';

const KEY = 'test-table';

describe('usePersistentColumnVisibility', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('stores only the hidden columns', () => {
    const { result } = renderHook(() => usePersistentColumnVisibility(KEY));

    act(() => {
      result.current[1]({ name: true, id: false });
    });

    expect(JSON.parse(localStorage.getItem(KEY) ?? '')).toEqual({ id: false });
  });

  it('forgets the hidden columns when visibility resets to empty', () => {
    localStorage.setItem(KEY, JSON.stringify({ id: false }));
    const { result } = renderHook(() => usePersistentColumnVisibility(KEY));

    act(() => {
      result.current[1]({});
    });

    expect(localStorage.getItem(KEY)).toBeNull();
  });
});
