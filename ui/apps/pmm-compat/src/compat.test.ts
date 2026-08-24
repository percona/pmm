jest.mock('@grafana/runtime', () => ({
  locationService: {
    getLocation: () => ({ pathname: '/', search: '', hash: '' }),
    push: jest.fn(),
    replace: jest.fn(),
  },
  getAppEvents: () => ({ subscribe: jest.fn() }),
  config: { bootData: { user: {} }, theme2: { isDark: true } },
  ThemeChangedEvent: class {},
}));
jest.mock('@grafana/data', () => ({
  BusEventBase: class {},
  textUtil: { sanitizeUrl: (url: string) => url },
  urlUtil: { appendQueryToUrl: (url: string) => url, toUrlParams: () => '' },
}));
jest.mock('@grafana/ui', () => ({}));

import { initialize } from './compat';

describe('compat', () => {
  const replaceMock = jest.fn();
  const reloadMock = jest.fn();
  const originalLocation = window.location;

  const setLocation = (search: string, pathname = '/graph/d/some-dashboard') => {
    Object.defineProperty(window, 'location', {
      value: {
        ...originalLocation,
        search,
        pathname,
        replace: replaceMock,
        reload: reloadMock,
      },
      writable: true,
    });
  };

  beforeEach(() => {
    replaceMock.mockClear();
    reloadMock.mockClear();
    // Keep the docked-nav keys unset so initialize() stops at its reload guard instead of
    // running on into the messenger setup, which needs more of @grafana/runtime than we mock.
    localStorage.clear();
  });

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('does not run compat logic when renderer is active (?render=1)', () => {
    setLocation('?render=1');

    initialize();

    expect(replaceMock).not.toHaveBeenCalled();
  });

  it('runs compat logic when render=0 (not renderer)', () => {
    setLocation('?render=0');

    initialize();

    expect(replaceMock).toHaveBeenCalled();
  });

  // These must stay in step with the nginx exclusion regex in
  // build/ansible/roles/nginx/files/conf.d/pmm.conf: nginx lets them through to Grafana, and the
  // plugin must not pull them into the shell afterwards.
  it.each([
    '/graph/api/datasources',
    '/graph/render/d/some-dashboard',
    '/graph/login',
    '/graph/login/generic_oauth',
    '/graph/logout',
    '/graph/signup',
    '/graph/invite/abc123',
    '/graph/verify',
    '/graph/user/password/send-reset-email',
    '/graph/user/password/reset',
  ])('does not redirect %s into the PMM UI', (pathname) => {
    setLocation('', pathname);

    initialize();

    expect(replaceMock).not.toHaveBeenCalled();
  });

  it.each(['/graph/d/some-dashboard', '/graph/apidocs', '/graph/logins', '/graph/'])(
    'redirects %s into the PMM UI',
    (pathname) => {
      setLocation('', pathname);

      initialize();

      expect(replaceMock).toHaveBeenCalled();
    }
  );
});
