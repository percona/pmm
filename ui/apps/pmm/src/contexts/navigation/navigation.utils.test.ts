import { addHighAvailability, addHomePage } from './navigation.utils';

describe('addHighAvailability', () => {
  it('is a single entry carrying the status badge and the namespace', () => {
    const item = addHighAvailability({
      enabled: true,
      health: 'degraded',
      nodes: [],
      namespace: 'pmm ha',
    });

    expect(item.children).toBeUndefined();
    expect(item.url).toBe(
      '/graph/d/pmm-ha-health-overview/pmm-ha-health-overview?var-namespace=pmm%20ha'
    );
    expect(item.matches).toEqual([
      '/graph/d/pmm-ha-health-overview/pmm-ha-health-overview',
    ]);
    expect(item.badge).toBeDefined();
    expect(item.badgeAlwaysVisible).toBe(true);
  });

  it('keeps the plain dashboard url without a namespace', () => {
    const item = addHighAvailability({
      enabled: true,
      health: 'healthy',
      nodes: [],
    });

    expect(item.url).toBe(
      '/graph/d/pmm-ha-health-overview/pmm-ha-health-overview'
    );
  });
});

describe('addHomePage', () => {
  it('also highlights a custom home dashboard', () => {
    expect(addHomePage({ homeDashboardUID: 'custom' }).matches).toContain(
      '/graph/d/custom'
    );
  });

  it('keeps the default entry without preferences', () => {
    expect(addHomePage().id).toBe('home-page');
  });
});
