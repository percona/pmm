import { describe, expect, it } from 'vitest';
import { mergeServiceIdFromSearchParams } from './qanTools';

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
