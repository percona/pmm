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

  it('reads the hidden columns of the new key when the key changes', () => {
    localStorage.setItem('table-a', JSON.stringify({ id: false }));
    localStorage.setItem('table-b', JSON.stringify({ name: false }));
    const { result, rerender } = renderHook(
      ({ key }) => usePersistentColumnVisibility(key),
      { initialProps: { key: 'table-a' } }
    );

    rerender({ key: 'table-b' });

    expect(result.current[0]).toEqual({ name: false });
    expect(JSON.parse(localStorage.getItem('table-a') ?? '')).toEqual({
      id: false,
    });
    expect(JSON.parse(localStorage.getItem('table-b') ?? '')).toEqual({
      name: false,
    });
  });

  it('keeps the hidden columns of the old key out of a new key', () => {
    localStorage.setItem('table-a', JSON.stringify({ id: false }));
    const { result, rerender } = renderHook(
      ({ key }) => usePersistentColumnVisibility(key),
      { initialProps: { key: 'table-a' } }
    );

    rerender({ key: 'table-b' });

    expect(result.current[0]).toEqual({});
    expect(localStorage.getItem('table-b')).toBeNull();
  });
});
