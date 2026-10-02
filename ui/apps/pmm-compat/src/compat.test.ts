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

import {
  GRAFANA_DIRECT_PATHS,
  GRAFANA_DIRECT_PATHS_OBFUSCATED,
  GRAFANA_SHELL_PATHS,
} from '@pmm/shared/fixtures';
import { initialize } from './compat';

describe('compat', () => {
  const replaceMock = jest.fn();
  const reloadMock = jest.fn();
  const originalLocation = window.location;

  const setLocation = (
    search: string,
    pathname = '/graph/d/some-dashboard'
  ) => {
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

  // nginx lets these through to Grafana, and the plugin must not pull them into the shell
  // afterwards. The lists live in @pmm/shared so nginx, the shell and this plugin cannot drift
  // apart; the obfuscated forms matter because nginx matches a decoded $uri.
  it.each([...GRAFANA_DIRECT_PATHS, ...GRAFANA_DIRECT_PATHS_OBFUSCATED])(
    'does not redirect %s into the PMM UI',
    (pathname) => {
      setLocation('', pathname);

      initialize();

      expect(replaceMock).not.toHaveBeenCalled();
    }
  );

  it.each(GRAFANA_SHELL_PATHS.filter((path) => path.startsWith('/graph/')))(
    'redirects %s into the PMM UI',
    (pathname) => {
      setLocation('', pathname);

      initialize();

      expect(replaceMock).toHaveBeenCalled();
    }
  );
});
