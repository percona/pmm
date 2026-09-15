import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { consumeReturnTo } from './auth.returnTo';
import { redirectToLogin } from './auth.utils';

describe('redirectToLogin', () => {
  const originalLocation = window.location;
  const replace = vi.fn();

  beforeEach(() => {
    replace.mockClear();
    Object.defineProperty(window, 'location', {
      value: {
        ...originalLocation,
        pathname: '/pmm-ui/graph/d/node-cpu',
        search: '',
        hash: '',
        replace,
      },
      writable: true,
    });
  });

  afterEach(() => {
    sessionStorage.clear();
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('remembers where the user was and sends them to the login page', () => {
    redirectToLogin();

    expect(replace).toHaveBeenCalledWith('/graph/login');
    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');
  });

  it('reaches the same verdict however many times a render calls it', () => {
    // Called once per render until the browser navigates, twice per pass under StrictMode.
    redirectToLogin();
    redirectToLogin();
    redirectToLogin();

    expect(consumeReturnTo()).toBe('/graph/d/node-cpu');
  });
});
