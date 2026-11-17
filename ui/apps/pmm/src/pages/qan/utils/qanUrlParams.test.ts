import { isQanDimensionFilterParam } from './qanUrlParams';

describe('qanUrlParams', () => {
  it('treats filter_by as reserved, not a dimension filter', () => {
    expect(isQanDimensionFilterParam('filter_by')).toBe(false);
    expect(isQanDimensionFilterParam('filter_service_name')).toBe(true);
    expect(isQanDimensionFilterParam('filter_service_id')).toBe(false);
  });
});
