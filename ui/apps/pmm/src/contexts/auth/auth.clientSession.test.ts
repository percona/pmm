import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import {
  clearClientSession,
  ensureClientSessionListener,
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

  it('notifies once when clearing with session listener installed', () => {
    ensureClientSessionListener();
    establishClientSession();
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent');

    clearClientSession();

    expect(dispatchSpy).toHaveBeenCalledTimes(1);
    dispatchSpy.mockRestore();
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

  describe('ensureClientSessionListener', () => {
    const sessionChangeEvent = expect.objectContaining({
      type: 'pmm-client-session-change',
    });

    beforeAll(() => {
      ensureClientSessionListener();
    });

    it('notifies on localStorage.clear()', () => {
      establishClientSession();
      const dispatchSpy = vi.spyOn(window, 'dispatchEvent');

      localStorage.clear();

      expect(dispatchSpy).toHaveBeenCalledWith(sessionChangeEvent);
      dispatchSpy.mockRestore();
    });

    it('notifies when removing a tracked key', () => {
      localStorage.setItem('pmm-ui.session.active', 'true');
      const dispatchSpy = vi.spyOn(window, 'dispatchEvent');

      localStorage.removeItem('pmm-ui.session.active');

      expect(dispatchSpy).toHaveBeenCalledWith(sessionChangeEvent);
      dispatchSpy.mockRestore();
    });

    it('notifies when removing a grafana-prefixed key', () => {
      localStorage.setItem('grafana.test', 'value');
      const dispatchSpy = vi.spyOn(window, 'dispatchEvent');

      localStorage.removeItem('grafana.test');

      expect(dispatchSpy).toHaveBeenCalledWith(sessionChangeEvent);
      dispatchSpy.mockRestore();
    });

    it('does not notify when removing an unrelated key', () => {
      localStorage.setItem('other-app.key', 'value');
      const dispatchSpy = vi.spyOn(window, 'dispatchEvent');

      localStorage.removeItem('other-app.key');

      expect(dispatchSpy).not.toHaveBeenCalled();
      dispatchSpy.mockRestore();
    });
  });
});
