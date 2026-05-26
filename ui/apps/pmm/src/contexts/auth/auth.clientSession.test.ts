import { afterEach, describe, expect, it } from 'vitest';
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

  it('is false after clearClientSession', () => {
    establishClientSession();
    clearClientSession();
    expect(isClientSessionEstablished()).toBe(false);
  });

  it('detects Grafana login paths', () => {
    expect(isGrafanaLoginPath('/graph/login')).toBe(true);
    expect(isGrafanaLoginPath('/graph/admin/users/edit/id')).toBe(false);
  });
});
