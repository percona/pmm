import { describe, it, expect } from '@jest/globals';
import { updateBodyClassByLocation } from './document';

describe('updateBodyClassByLocation', () => {
  it('should add the correct class to the body', () => {
    const location = {
      pathname: '/graph/123',
    };
    updateBodyClassByLocation(location as Location);

    expect(document.body.classList.contains('grafana-compat-page-123')).toBe(true);
  });

  it('should remove the previous class from the body', () => {
    const location = {
      pathname: '/graph/123',
    };
    updateBodyClassByLocation(location as Location);

    expect(document.body.classList.contains('grafana-compat-page-123')).toBe(true);

    const location2 = {
      pathname: '/graph/456',
    };
    updateBodyClassByLocation(location2 as Location);

    expect(document.body.classList.contains('grafana-compat-page-456')).toBe(true);
    expect(document.body.classList.contains('grafana-compat-page-123')).toBe(false);
  });
});
