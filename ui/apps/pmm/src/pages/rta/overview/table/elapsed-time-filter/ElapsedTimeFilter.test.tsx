import { fireEvent, render, screen } from '@testing-library/react';
import type { MRT_Column } from 'material-react-table';
import type { QueryData } from 'types/rta.types';
import ElapsedTimeFilter from './ElapsedTimeFilter';

// The filter value normally lives in the table, and an external change to it
// (URL back/forward, a pasted link) is what these tests stand in for.
const stubColumn = (initial: [string, string] = ['', '']) => {
  let filterValue: unknown = initial;
  return {
    column: {
      getFilterValue: () => filterValue,
      setFilterValue: (updater: unknown) => {
        filterValue =
          typeof updater === 'function' ? updater(filterValue) : updater;
      },
    } as unknown as MRT_Column<QueryData>,
    range: () => filterValue as [string, string],
  };
};

const minField = () => screen.getByLabelText('Min') as HTMLInputElement;

describe('ElapsedTimeFilter', () => {
  it('clears the field when an external change brings an unusable bound', () => {
    const { column, range } = stubColumn();
    const { rerender } = render(<ElapsedTimeFilter column={column} />);

    fireEvent.change(minField(), { target: { value: '10' } });
    expect(minField().value).toBe('10');

    column.setFilterValue(['abc', '']);
    rerender(<ElapsedTimeFilter column={column} />);

    expect(minField().value).toBe('');
    expect(range()).toEqual(['', '']);
  });

  it('takes an external change that is a usable bound', () => {
    const { column } = stubColumn();
    const { rerender } = render(<ElapsedTimeFilter column={column} />);

    column.setFilterValue(['20', '']);
    rerender(<ElapsedTimeFilter column={column} />);

    expect(minField().value).toBe('20');
  });

  it('lets a lone decimal point be typed without filtering on it', () => {
    const { column, range } = stubColumn();
    render(<ElapsedTimeFilter column={column} />);

    fireEvent.change(minField(), { target: { value: '.' } });

    expect(minField().value).toBe('.');
    expect(range()).toEqual(['', '']);

    fireEvent.change(minField(), { target: { value: '.5' } });

    expect(minField().value).toBe('.5');
    expect(range()).toEqual(['.5', '']);
  });
});
