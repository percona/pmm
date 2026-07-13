import { orderByFromSorting, sortingFromOrderBy } from './qanOrderBy';

describe('qanOrderBy', () => {
  it('maps API DESC prefix to MRT descending sort', () => {
    expect(sortingFromOrderBy('-load')).toEqual([{ id: 'load', desc: true }]);
  });

  it('maps API ASC (no prefix) to MRT ascending sort', () => {
    expect(sortingFromOrderBy('load')).toEqual([{ id: 'load', desc: false }]);
  });

  it('encodes MRT descending sort as API order_by with minus prefix', () => {
    expect(orderByFromSorting([{ id: 'query_time', desc: true }])).toBe('-query_time');
  });

  it('encodes MRT ascending sort as API order_by without prefix', () => {
    expect(orderByFromSorting([{ id: 'query_time', desc: false }])).toBe('query_time');
  });
});
