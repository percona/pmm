import { renderHook } from '@testing-library/react';
import { QueryData } from 'types/rta.types';
import { TEST_MONGO_DB_QUERY_DATA } from 'utils/testStubs';
import { useStatementNavigation } from './useStatementNavigation';

const row = (queryId: string): QueryData => ({
  ...TEST_MONGO_DB_QUERY_DATA,
  queryId,
});

const a = row('a');
const b = row('b');
const c = row('c');

const setup = (rows: QueryData[], selected: QueryData | undefined) => {
  const onSelect = vi.fn();
  const hook = renderHook(
    ({ rows, selected }) =>
      useStatementNavigation({ rows, selected, onSelect }),
    { initialProps: { rows, selected } }
  );

  return { ...hook, onSelect };
};

describe('useStatementNavigation', () => {
  it('navigates the list while the selected statement is in it', () => {
    const { result, onSelect } = setup([a, b, c], b);

    expect(result.current.isFirst).toBe(false);
    expect(result.current.isLast).toBe(false);

    result.current.next();
    expect(onSelect).toHaveBeenLastCalledWith(c);
    result.current.previous();
    expect(onSelect).toHaveBeenLastCalledWith(a);
  });

  it('moves to the neighbours of the position a finished statement had', () => {
    const { result, rerender, onSelect } = setup([a, b, c], b);

    rerender({ rows: [a, c], selected: b });

    expect(result.current.isFirst).toBe(false);
    expect(result.current.isLast).toBe(false);
    result.current.next();
    expect(onSelect).toHaveBeenLastCalledWith(c);
    result.current.previous();
    expect(onSelect).toHaveBeenLastCalledWith(a);
  });

  it('offers only previous when the rows after it have finished too', () => {
    const { result, rerender, onSelect } = setup([a, b, c], b);

    rerender({ rows: [a], selected: b });

    expect(result.current.isFirst).toBe(false);
    expect(result.current.isLast).toBe(true);
    result.current.next();
    expect(onSelect).not.toHaveBeenCalled();
    result.current.previous();
    expect(onSelect).toHaveBeenLastCalledWith(a);
  });

  it('offers nothing once every row has gone', () => {
    const { result, rerender } = setup([a, b, c], b);

    rerender({ rows: [], selected: b });

    expect(result.current.isFirst).toBe(true);
    expect(result.current.isLast).toBe(true);
  });

  it('offers nothing with the pane closed', () => {
    const { result } = setup([a, b, c], undefined);

    expect(result.current.isFirst).toBe(true);
    expect(result.current.isLast).toBe(true);
  });
});
