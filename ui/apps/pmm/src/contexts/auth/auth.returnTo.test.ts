import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  consumeReturnTo,
  isRestorableReturnTo,
  saveReturnTo,
} from './auth.returnTo';
import {
  AUTH_RETURN_TO_LOOP_WINDOW_MS,
  AUTH_RETURN_TO_TTL_MS,
} from 'lib/constants';
import { GRAFANA_DIRECT_PATHS } from '@pmm/shared/fixtures';

const RETURN_TO_KEY = 'pmm-ui.auth.returnTo';
const GRAFANA_REDIRECT_TO_KEY = 'redirectTo';

const at = (pathname: string, search = '', hash = '') => ({
  pathname,
  search,
  hash,
});

describe('auth.returnTo', () => {
  const originalLocation = window.location;
  const originalSessionStorage = window.sessionStorage;

  afterEach(() => {
    sessionStorage.clear();
    vi.useRealTimers();
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('saves the shell-relative path with query and hash', () => {
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu', '?viewPanel=22', '#panel-22'));

    expect(consumeReturnTo()).toBe('/graph/d/node-cpu?viewPanel=22#panel-22');
  });

  it('saves native PMM UI pages, not just Grafana routes', () => {
    saveReturnTo(at('/pmm-ui/settings/advanced'));

    expect(consumeReturnTo()).toBe('/settings/advanced');
  });

  it('keeps a later /pmm-ui segment intact', () => {
    saveReturnTo(at('/pmm-ui/graph/d/pmm-ui-dashboard'));

    expect(consumeReturnTo()).toBe('/graph/d/pmm-ui-dashboard');
  });

  it.each(['/pmm-ui', '/pmm-ui/', '/pmm-ui/graph', '/pmm-ui/graph/'])(
    'does not save %s — nothing to restore',
    (pathname) => {
      saveReturnTo(at(pathname));

      expect(consumeReturnTo()).toBeNull();
    }
  );

  // Shared with nginx and the compat plugin via @pmm/shared, so the three cannot drift apart.
  it.each(GRAFANA_DIRECT_PATHS.map((path) => `/pmm-ui${path}`))(
    'does not save Grafana direct route %s',
    (pathname) => {
      saveReturnTo(at(pathname));

      expect(consumeReturnTo()).toBeNull();
    }
  );

  it('does not save while the image renderer is driving', () => {
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, search: '?render=1' },
      writable: true,
    });

    saveReturnTo(at('/pmm-ui/graph/d/node-cpu', '?render=1'));

    expect(consumeReturnTo()).toBeNull();
  });

  it("drops Grafana's own redirectTo on save and on consume", () => {
    sessionStorage.setItem(GRAFANA_REDIRECT_TO_KEY, '/d/somewhere-else');
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));
    expect(sessionStorage.getItem(GRAFANA_REDIRECT_TO_KEY)).toBeNull();

    sessionStorage.setItem(GRAFANA_REDIRECT_TO_KEY, '/d/somewhere-else');
    consumeReturnTo();
    expect(sessionStorage.getItem(GRAFANA_REDIRECT_TO_KEY)).toBeNull();
  });

  it('returns the target once and only once', () => {
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));

    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');
    expect(consumeReturnTo()).toBeNull();
  });

  it('returns null when nothing was saved', () => {
    expect(consumeReturnTo()).toBeNull();
  });

  it('expires a target past the TTL', () => {
    vi.useFakeTimers();
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));

    vi.advanceTimersByTime(AUTH_RETURN_TO_TTL_MS + 1);

    expect(consumeReturnTo()).toBeNull();
  });

  it('keeps a target that is still inside the TTL', () => {
    vi.useFakeTimers();
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));

    vi.advanceTimersByTime(AUTH_RETURN_TO_TTL_MS - 1);

    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');
  });

  it.each([
    '//evil.com',
    'https://evil.com/x',
    '/graph/../../etc/passwd',
    'graph/d/node-cpu',
    '/graph\\evil',
    '/graph/login',
    '/graph',
    '/graph?orgId=1',
    // Only reveal their shape once decoded, so the raw same-origin checks cannot see them.
    '/%2F%2Fevil.com',
    '/%5Cevil.com',
    '/%2e%2e/x',
    // Malformed escapes have no single meaning, so there is nothing safe to classify.
    '/graph/d/%zz',
    '/graph/d/%',
    // Percent-encoded and slash-padded Grafana routes: nginx exempts these, so must we.
    '/graph/%6Cogin',
    '/graph//login',
  ])('rejects %s as a restorable target', (target) => {
    expect(isRestorableReturnTo(target)).toBe(false);
  });

  it.each([
    '/graph/d/node-cpu?viewPanel=22',
    '/settings/advanced',
    '/help',
    // '..' is legitimate inside a Grafana template variable value
    '/graph/d/node-cpu?var-version=1..2',
    '/graph/d/node-cpu#panel..22',
  ])('accepts %s as a restorable target', (target) => {
    expect(isRestorableReturnTo(target)).toBe(true);
  });

  it('round-trips a target whose query contains ..', () => {
    saveReturnTo(
      at('/pmm-ui/graph/d/node-cpu', '?viewPanel=22&var-version=1..2')
    );

    expect(consumeReturnTo()).toBe(
      '/graph/d/node-cpu?viewPanel=22&var-version=1..2'
    );
  });

  it('does not save a percent-encoded Grafana auth route', () => {
    saveReturnTo(at('/pmm-ui/graph/%6Cogin'));

    expect(consumeReturnTo()).toBeNull();
  });

  it('refuses a poisoned stored target', () => {
    sessionStorage.setItem(
      RETURN_TO_KEY,
      JSON.stringify({ path: '//evil.com', at: Date.now() })
    );

    expect(consumeReturnTo()).toBeNull();
  });

  it('refuses a stored value that is not JSON', () => {
    sessionStorage.setItem(RETURN_TO_KEY, 'not json at all');

    expect(consumeReturnTo()).toBeNull();
  });

  describe('when sessionStorage is unavailable', () => {
    // Blocked-site-data policies throw on the accessor itself, not just on setItem.
    const breakStorage = () => {
      Object.defineProperty(window, 'sessionStorage', {
        get() {
          throw new DOMException('The operation is insecure.', 'SecurityError');
        },
        configurable: true,
      });
    };

    afterEach(() => {
      Object.defineProperty(window, 'sessionStorage', {
        value: originalSessionStorage,
        configurable: true,
        writable: true,
      });
    });

    it('does not throw while saving', () => {
      breakStorage();

      expect(() => saveReturnTo(at('/pmm-ui/graph/d/node-cpu'))).not.toThrow();
    });

    it('does not throw while consuming, and restores nothing', () => {
      breakStorage();

      expect(() => consumeReturnTo()).not.toThrow();
      expect(consumeReturnTo()).toBeNull();
    });
  });

  it('does not re-save the target it just restored', () => {
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));
    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');

    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));

    expect(consumeReturnTo()).toBeNull();
  });

  it('remembers the restored page again once the loop window has passed', () => {
    // Restore a deep link, work there a while, then have the session expire on that same page.
    vi.useFakeTimers();
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));
    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');

    vi.advanceTimersByTime(AUTH_RETURN_TO_LOOP_WINDOW_MS + 1);
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));

    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');
  });

  it('still refuses an immediate bounce back off the restored page', () => {
    vi.useFakeTimers();
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));
    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');

    vi.advanceTimersByTime(AUTH_RETURN_TO_LOOP_WINDOW_MS - 1);
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));

    expect(consumeReturnTo()).toBeNull();
  });

  it('still saves a different target after a restore', () => {
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));
    consumeReturnTo();

    saveReturnTo(at('/pmm-ui/graph/d/node-memory'));

    expect(consumeReturnTo()).toBe('/graph/d/node-memory');
  });

  it('reaches the same verdict however many times it is called', () => {
    // AuthProvider calls this once per render, so the verdict must not depend on the render count.
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));
    consumeReturnTo();

    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));

    expect(consumeReturnTo()).toBeNull();
  });
});
