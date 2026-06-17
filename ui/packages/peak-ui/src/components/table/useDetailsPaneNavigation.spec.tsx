import { renderHook, act } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useDetailsPaneNavigation } from './useDetailsPaneNavigation';

type Row = { id: string; name: string };

const rows: Row[] = [
  { id: 'a', name: 'Alpha' },
  { id: 'b', name: 'Beta' },
  { id: 'c', name: 'Gamma' },
];

const getRowId = (row: Row) => row.id;

describe('useDetailsPaneNavigation', () => {
  it('returns index -1 when selected is undefined', () => {
    const { result } = renderHook(() =>
      useDetailsPaneNavigation({
        rows,
        selected: undefined,
        getRowId,
        onSelect: vi.fn(),
      })
    );

    expect(result.current.index).toBe(-1);
    expect(result.current.isFirst).toBe(true);
    expect(result.current.isLast).toBe(true);
    expect(result.current.hasPrevious).toBe(false);
    expect(result.current.hasNext).toBe(false);
  });

  it('returns index -1 when selection is missing from rows', () => {
    const { result } = renderHook(() =>
      useDetailsPaneNavigation({
        rows,
        selected: { id: 'missing', name: 'Missing' },
        getRowId,
        onSelect: vi.fn(),
      })
    );

    expect(result.current.index).toBe(-1);
    expect(result.current.hasPrevious).toBe(false);
    expect(result.current.hasNext).toBe(false);
  });

  it('marks first row as isFirst and disables previous', () => {
    const { result } = renderHook(() =>
      useDetailsPaneNavigation({
        rows,
        selected: rows[0],
        getRowId,
        onSelect: vi.fn(),
      })
    );

    expect(result.current.index).toBe(0);
    expect(result.current.isFirst).toBe(true);
    expect(result.current.isLast).toBe(false);
    expect(result.current.hasPrevious).toBe(false);
    expect(result.current.hasNext).toBe(true);
  });

  it('marks last row as isLast and disables next', () => {
    const { result } = renderHook(() =>
      useDetailsPaneNavigation({
        rows,
        selected: rows[2],
        getRowId,
        onSelect: vi.fn(),
      })
    );

    expect(result.current.index).toBe(2);
    expect(result.current.isFirst).toBe(false);
    expect(result.current.isLast).toBe(true);
    expect(result.current.hasPrevious).toBe(true);
    expect(result.current.hasNext).toBe(false);
  });

  it('marks middle row with both directions available', () => {
    const { result } = renderHook(() =>
      useDetailsPaneNavigation({
        rows,
        selected: rows[1],
        getRowId,
        onSelect: vi.fn(),
      })
    );

    expect(result.current.index).toBe(1);
    expect(result.current.isFirst).toBe(false);
    expect(result.current.isLast).toBe(false);
    expect(result.current.hasPrevious).toBe(true);
    expect(result.current.hasNext).toBe(true);
  });

  it('calls onSelect with the next row', () => {
    const onSelect = vi.fn();
    const { result } = renderHook(() =>
      useDetailsPaneNavigation({
        rows,
        selected: rows[0],
        getRowId,
        onSelect,
      })
    );

    act(() => {
      result.current.next();
    });

    expect(onSelect).toHaveBeenCalledWith(rows[1]);
  });

  it('calls onSelect with the previous row', () => {
    const onSelect = vi.fn();
    const { result } = renderHook(() =>
      useDetailsPaneNavigation({
        rows,
        selected: rows[2],
        getRowId,
        onSelect,
      })
    );

    act(() => {
      result.current.previous();
    });

    expect(onSelect).toHaveBeenCalledWith(rows[1]);
  });

  it('does not call onSelect at boundaries', () => {
    const onSelect = vi.fn();
    const { result } = renderHook(() =>
      useDetailsPaneNavigation({
        rows,
        selected: rows[0],
        getRowId,
        onSelect,
      })
    );

    act(() => {
      result.current.previous();
    });

    expect(onSelect).not.toHaveBeenCalled();
  });

  it('does not navigate when index is -1', () => {
    const onSelect = vi.fn();
    const { result } = renderHook(() =>
      useDetailsPaneNavigation({
        rows,
        selected: undefined,
        getRowId,
        onSelect,
      })
    );

    act(() => {
      result.current.next();
      result.current.previous();
    });

    expect(onSelect).not.toHaveBeenCalled();
  });
});
