import { describe, it, expect } from 'vitest';
import { keepTimeRange } from './link.utils';

const dashboard = {
  pathname: '/graph/d/mysql-instance-summary/mysql-instance-summary',
  search: '?from=now-1h&to=now&timezone=utc&var-service_name=mysql-1',
};

describe('keepTimeRange', () => {
  it('should carry the current time range over to a dashboard link', () => {
    expect(keepTimeRange('/graph/d/node-cpu/cpu-utilization', dashboard)).toBe(
      '/graph/d/node-cpu/cpu-utilization?from=now-1h&to=now&timezone=utc'
    );
  });

  it('should append to the params the link already has', () => {
    expect(keepTimeRange('/graph/d/node-cpu?orgId=1', dashboard)).toBe(
      '/graph/d/node-cpu?orgId=1&from=now-1h&to=now&timezone=utc'
    );
  });

  it('should keep a time range the link already sets', () => {
    const url = '/graph/d/node-cpu?from=now-6h&to=now&timezone=browser';
    expect(keepTimeRange(url, dashboard)).toBe(url);
  });

  it('should leave the link as is when the current page has no time range', () => {
    expect(
      keepTimeRange('/graph/d/node-cpu', { ...dashboard, search: '' })
    ).toBe('/graph/d/node-cpu');
  });

  it('should leave the link as is when the current page is not a dashboard', () => {
    expect(
      keepTimeRange('/graph/d/node-cpu', {
        pathname: '/graph/explore',
        search: '?from=now-1h&to=now',
      })
    ).toBe('/graph/d/node-cpu');
  });

  it('should leave non-dashboard links as is', () => {
    expect(keepTimeRange('/help', dashboard)).toBe('/help');
  });
});
