// Mock Grafana modules to avoid loading ESM-only deps in Jest
jest.mock('@grafana/data', () => ({
  DataLinkBuiltInVars: { keepTime: 'keepTime', includeVars: 'includeVars' },
  locationUtil: { assureBaseUrl: (url: string) => url },
  textUtil: { sanitizeUrl: (url: string) => url },
  urlUtil: {
    appendQueryToUrl: (url: string, _params: string) => url,
    toUrlParams: () => '',
  },
}));
const mockReplace = jest.fn((url: string) => url);
jest.mock('@grafana/runtime', () => ({
  config: { disableSanitizeHtml: false },
  getTemplateSrv: () => ({ replace: mockReplace }),
}));

import {
  cleanupVariables,
  getLinkWithVariables,
  shouldIncludeVars,
  waitForDashboardScene,
} from './variables';

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

  describe('on a dashboard', () => {
    const sceneWindow = window as {
      __grafanaSceneContext?: { isActive: boolean };
    };

    beforeEach(() => {
      mockLocation(dashboards.pg);
      mockReplace.mockClear();
    });

    afterEach(() => {
      delete sceneWindow.__grafanaSceneContext;
    });

    it('should not expand variables before the dashboard scene is active', () => {
      expect(getLinkWithVariables(dashboards.pgSummary)).toBe(
        dashboards.pgSummary
      );
      expect(mockReplace).not.toHaveBeenCalled();
    });

    it('should expand variables once the dashboard scene is active', () => {
      sceneWindow.__grafanaSceneContext = { isActive: true };
      getLinkWithVariables(dashboards.pgSummary);
      expect(mockReplace).toHaveBeenCalled();
    });
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

describe('waitForDashboardScene', () => {
  const sceneWindow = window as {
    __grafanaSceneContext?: { isActive: boolean };
  };

  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
    delete sceneWindow.__grafanaSceneContext;
  });

  const isSettled = async (promise: Promise<void>) => {
    let settled = false;
    promise.then(() => (settled = true));
    await jest.advanceTimersByTimeAsync(0);
    return settled;
  };

  it('should resolve immediately when not on a dashboard', async () => {
    mockLocation('/alerting/list');
    expect(await isSettled(waitForDashboardScene())).toBe(true);
  });

  it('should wait until the dashboard scene is active', async () => {
    mockLocation(dashboards.pg);
    const promise = waitForDashboardScene();

    await jest.advanceTimersByTimeAsync(1_000);
    expect(await isSettled(promise)).toBe(false);

    sceneWindow.__grafanaSceneContext = { isActive: true };
    await jest.advanceTimersByTimeAsync(50);
    expect(await isSettled(promise)).toBe(true);
  });

  it('should give up when the dashboard scene never activates', async () => {
    mockLocation(dashboards.pg);
    const promise = waitForDashboardScene();

    await jest.advanceTimersByTimeAsync(5_000);
    expect(await isSettled(promise)).toBe(true);
  });
});

describe('cleanupVariables', () => {
  it("should return the same url if it doesn't have variables", () => {
    const url =
      'https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview';
    const result = cleanupVariables(url);
    expect(result).toBe(url);
  });

  it('should return the url with the variables empty variables removed', () => {
    const url =
      'https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-empty=&var-empty-old=None&var-value=Value';
    const expected =
      'https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-value=Value';
    const result = cleanupVariables(url);
    expect(result).toBe(expected);
  });

  it('should return the url with the variables with the All value removed', () => {
    const url =
      'https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-all=$__all&val-all-old=All&var-value=Value';
    const expected =
      'https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-value=Value';
    const result = cleanupVariables(url);
    expect(result).toBe(expected);
  });

  it('should return the url with the variables with all and no value removed', () => {
    const url =
      'https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-all=$__all&val-all-old=All&var-empty=&var-empty-old=None&var-value=Value';
    const expected =
      'https://percona.com/d/postgresql-instance-overview/postgresql-instances-overview?var-value=Value';
    const result = cleanupVariables(url);
    expect(result).toBe(expected);
  });
});
