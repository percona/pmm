import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { getLinkWithVariables, shouldIncludeVars } from './variables';

const prefixes = {
  grafana: '/graph',
  pmm: '/pmm-ui/next',
};

const dashboards = {
  pg: '/d/postgresql-instance-overview/postgresql-instances-overview',
  pgSummary: '/d/postgresql-instance-summary/postgresql-instance-summary',
  mysql: '/d/mysql-instance-summary/mysql-instance-summary',
  node: '/d/node-overview/node-overview',
};

const mockLocation = (pathname: string) => {
  Object.defineProperty(window, 'location', {
    value: {
      pathname,
      origin: 'https://percona.com',
    },
    writable: true,
  });
};

describe('getLinkWithVariables', () => {
  beforeEach(() => {
    mockLocation('/percona.com');
  });

  it('should return the same url if it is not a dashboard url', () => {
    const url = 'https://percona.com';
    const result = getLinkWithVariables(url);
    expect(result).toBe(url);
  });
});

describe('shouldIncludeVars', () => {
  it('should handle different prefixes', () => {
    const urls = [
      dashboards.pg,
      `${prefixes.grafana}${dashboards.pg}`,
      `${prefixes.pmm}${prefixes.grafana}${dashboards.pg}`,
    ];

    urls.forEach((url) => {
      mockLocation(dashboards.pg);
      const result = shouldIncludeVars(url);
      expect(result).toBe(true);
    });
  });

  it('should return true if the db type matches the current one', () => {
    mockLocation(dashboards.pg);
    const result = shouldIncludeVars(dashboards.pgSummary);
    expect(result).toBe(true);
  });

  it('should return true if the target db type is node', () => {
    mockLocation(dashboards.pg);
    const result = shouldIncludeVars(dashboards.node);
    expect(result).toBe(true);
  });

  it('should return false if the target db type is not the same as the current one', () => {
    mockLocation(dashboards.pg);
    const result = shouldIncludeVars(dashboards.mysql);
    expect(result).toBe(false);
  });

  it('should return false if current db type is node and target db type is not node', () => {
    mockLocation(dashboards.node);
    const result = shouldIncludeVars(dashboards.pg);
    expect(result).toBe(false);
  });
});
