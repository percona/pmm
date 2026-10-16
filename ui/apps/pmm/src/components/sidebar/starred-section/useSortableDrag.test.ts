import { findDragTarget } from './useSortableDrag';

const ROWS = [
  { top: 0, height: 40 },
  { top: 40, height: 40 },
  { top: 80, height: 40 },
];

describe('findDragTarget', () => {
  it('lets the middle row take the last place when dragged all the way down', () => {
    expect(findDragTarget(ROWS, 1, 200)).toEqual({ offset: 40, to: 2 });
  });

  it('lets the middle row take the first place when dragged all the way up', () => {
    expect(findDragTarget(ROWS, 1, -200)).toEqual({ offset: -40, to: 0 });
  });

  it('keeps a row in place until its centre reaches the next one', () => {
    expect(findDragTarget(ROWS, 1, 39).to).toBe(1);
    expect(findDragTarget(ROWS, 1, -39).to).toBe(1);
  });

  it('moves the first row to the end in one drag', () => {
    expect(findDragTarget(ROWS, 0, 500)).toEqual({ offset: 80, to: 2 });
  });

  it('reaches the end past a taller last row', () => {
    const rows = [...ROWS.slice(0, 2), { top: 80, height: 52 }];

    expect(findDragTarget(rows, 1, 500).to).toBe(2);
  });
});
