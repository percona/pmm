import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  clearClientSession,
  establishClientSession,
  isClientSessionEstablished,
  isGrafanaLoginPath,
} from './auth.clientSession';

describe('auth.clientSession', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('is false before establishClientSession', () => {
    expect(isClientSessionEstablished()).toBe(false);
  });

  it('is true after establishClientSession', () => {
    establishClientSession();
    expect(isClientSessionEstablished()).toBe(true);
  });

  it('does not notify when session is already established', () => {
    establishClientSession();
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent');

    establishClientSession();

    expect(dispatchSpy).not.toHaveBeenCalled();
    dispatchSpy.mockRestore();
  });

  it('is false after clearClientSession', () => {
    establishClientSession();
    clearClientSession();
    expect(isClientSessionEstablished()).toBe(false);
  });

  it('does not notify when session is already cleared', () => {
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent');

    clearClientSession();

    expect(dispatchSpy).not.toHaveBeenCalled();
    dispatchSpy.mockRestore();
  });

  it('detects Grafana login paths', () => {
    expect(isGrafanaLoginPath('/graph/login')).toBe(true);
    expect(isGrafanaLoginPath('/graph/admin/users/edit/id')).toBe(false);
  });
});
