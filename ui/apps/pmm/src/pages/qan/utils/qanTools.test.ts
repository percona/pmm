import { mergeServiceIdFromSearchParams, labelsFromSearchParams } from './qanTools';

describe('mergeServiceIdFromSearchParams', () => {
  it('maps service_id query param to service_id label', () => {
    const params = new URLSearchParams('service_id=abc-123');
    const labels = mergeServiceIdFromSearchParams({}, params);
    expect(labels.service_id).toEqual(['abc-123']);
  });

  it('strips /service_id/ prefix', () => {
    const params = new URLSearchParams('filter_service_id=%2Fservice_id%2Fuuid-1');
    const labels = mergeServiceIdFromSearchParams({}, params);
    expect(labels.service_id).toEqual(['uuid-1']);
  });
});

describe('labelsFromSearchParams', () => {
  it('does not treat filter_by as a dimension label', () => {
    const params = new URLSearchParams('filter_by=query-uuid&filter_service_name=mysql1');
    const labels = labelsFromSearchParams(params);
    expect(labels.by).toBeUndefined();
    expect(labels.service_name).toEqual(['mysql1']);
  });

  it('ignores filter_service_id as dimension (handled separately)', () => {
    const params = new URLSearchParams('filter_service_id=uuid-1');
    expect(labelsFromSearchParams(params).service_id).toBeUndefined();
  });
});
