import { readRange } from './ElapsedTimeFilter.utils';

describe('readRange', () => {
  it('reads both bounds', () => {
    expect(readRange(['1', '2'])).toEqual(['1', '2']);
  });

  it('fills in a missing bound', () => {
    expect(readRange(['1'])).toEqual(['1', '']);
    expect(readRange([null, '2'])).toEqual(['', '2']);
    expect(readRange([undefined, undefined])).toEqual(['', '']);
  });

  it('turns numbers into strings', () => {
    expect(readRange([1, 2.5])).toEqual(['1', '2.5']);
  });

  it('falls back to an empty range for anything that is not an array', () => {
    expect(readRange(undefined)).toEqual(['', '']);
    expect(readRange(null)).toEqual(['', '']);
    expect(readRange('1,2')).toEqual(['', '']);
  });
});
