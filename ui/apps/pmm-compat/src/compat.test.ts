jest.mock('@grafana/runtime', () => ({
  locationService: { getLocation: () => ({ pathname: '/', search: '', hash: '' }), push: jest.fn(), replace: jest.fn() },
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
  const originalLocation = window.location;

  const setLocation = (search: string, pathname = '/graph/d/some-dashboard') => {
    Object.defineProperty(window, 'location', {
      value: {
        ...originalLocation,
        search,
        pathname,
        replace: replaceMock,
      },
      writable: true,
    });
  };

  beforeEach(() => {
    replaceMock.mockClear();
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
});
