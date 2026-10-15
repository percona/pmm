import { splitSeverities, sumSeverities } from './CheckPanel.utils';

const counts = {
  emergency: 1,
  alert: 2,
  critical: 3,
  error: 4,
  warning: 5,
  notice: 6,
  info: 7,
  debug: 8,
};

describe('CheckPanel utils', () => {
  it('should add up counts of all services', () => {
    expect(
      splitSeverities([
        { serviceName: 'a', serviceId: '1', counts },
        { serviceName: 'b', serviceId: '2', counts },
      ])
    ).toEqual({
      emergency: 2,
      alert: 4,
      critical: 6,
      error: 8,
      warning: 10,
      notice: 12,
      info: 14,
      debug: 16,
    });
  });

  it('should return zero counts without services', () => {
    expect(sumSeverities(splitSeverities())).toBe(0);
  });

  it('should sum all severities', () => {
    expect(sumSeverities(counts)).toBe(36);
  });
});
