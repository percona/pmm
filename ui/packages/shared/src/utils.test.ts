import { isRenderingServer } from './utils';

describe('isRenderingServer', () => {
  const originalLocation = window.location;

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('returns true when URL has render=1 (Grafana image renderer)', () => {
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, search: '?render=1' },
      writable: true,
    });

    expect(isRenderingServer()).toBe(true);
  });

  it('returns true when render=1 is among other params', () => {
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, search: '?foo=bar&render=1&baz=qux' },
      writable: true,
    });

    expect(isRenderingServer()).toBe(true);
  });

  it('returns false when search is empty', () => {
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, search: '' },
      writable: true,
    });

    expect(isRenderingServer()).toBe(false);
  });

  it('returns false when render has a value other than 1', () => {
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, search: '?render=0' },
      writable: true,
    });

    expect(isRenderingServer()).toBe(false);
  });
});
