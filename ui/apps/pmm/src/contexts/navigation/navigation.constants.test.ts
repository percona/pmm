import { afterEach, describe, expect, it } from 'vitest';
import { NAV_SIGN_IN, NAV_SIGN_OUT } from './navigation.constants';
import { consumeReturnTo } from 'contexts/auth/auth.returnTo';

describe('sign in / sign out', () => {
  const originalLocation = window.location;

  const atPath = (pathname: string) => {
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, pathname, search: '', hash: '' },
      writable: true,
    });
  };

  afterEach(() => {
    sessionStorage.clear();
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('remembers the current page when the user signs in explicitly', () => {
    // The anchor bypasses redirectToLogin(), which is where this would otherwise be recorded.
    atPath('/pmm-ui/graph/d/node-cpu');

    NAV_SIGN_IN.onClick?.();

    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');
  });

  it('stays an anchor so /graph/login is reached as a document load', () => {
    expect(NAV_SIGN_IN.url).toBe('/graph/login');
    expect(NAV_SIGN_IN.target).toBe('_self');
  });

  it('remembers the page even moments after the shell restored the user to it', () => {
    // A restore arms the bounce guard; a deliberate click on that same page must still register.
    atPath('/pmm-ui/graph/d/node-cpu');
    NAV_SIGN_IN.onClick?.();
    consumeReturnTo();

    NAV_SIGN_IN.onClick?.();

    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');
  });

  it('forgets the page when the user signs out', () => {
    atPath('/pmm-ui/graph/d/node-cpu');
    NAV_SIGN_IN.onClick?.();

    NAV_SIGN_OUT.onClick?.();

    expect(consumeReturnTo()).toBeNull();
  });
});
