import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  consumeReturnTo,
  isRestorableReturnTo,
  saveReturnTo,
} from './auth.returnTo';
import { AUTH_RETURN_TO_TTL_MS } from 'lib/constants';

const RETURN_TO_KEY = 'pmm-ui.auth.returnTo';
const GRAFANA_REDIRECT_TO_KEY = 'redirectTo';

const at = (pathname: string, search = '', hash = '') => ({
  pathname,
  search,
  hash,
});

describe('auth.returnTo', () => {
  const originalLocation = window.location;

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

  it.each([
    '/pmm-ui/graph/login',
    '/pmm-ui/graph/logout',
    '/pmm-ui/graph/api/datasources',
    '/pmm-ui/graph/render/d/x',
    '/pmm-ui/graph/signup',
    '/pmm-ui/graph/invite/abc123',
    '/pmm-ui/graph/verify',
    '/pmm-ui/graph/user/password/send-reset-email',
    '/pmm-ui/graph/user/password/reset',
  ])('does not save Grafana direct route %s', (pathname) => {
    saveReturnTo(at(pathname));

    expect(consumeReturnTo()).toBeNull();
  });

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
  ])('rejects %s as a restorable target', (target) => {
    expect(isRestorableReturnTo(target)).toBe(false);
  });

  it.each(['/graph/d/node-cpu?viewPanel=22', '/settings/advanced', '/help'])(
    'accepts %s as a restorable target',
    (target) => {
      expect(isRestorableReturnTo(target)).toBe(true);
    }
  );

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

  it('does not re-save the target it just restored', () => {
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));
    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');

    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));

    expect(consumeReturnTo()).toBeNull();
  });

  it('still saves a different target after a restore', () => {
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));
    consumeReturnTo();

    saveReturnTo(at('/pmm-ui/graph/d/node-memory'));

    expect(consumeReturnTo()).toBe('/graph/d/node-memory');
  });

  it('clears the loop guard once it has refused, so a later visit works again', () => {
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));
    consumeReturnTo();
    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));

    saveReturnTo(at('/pmm-ui/graph/d/node-cpu'));

    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');
  });
});
