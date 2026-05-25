import { afterEach, describe, expect, it } from 'vitest';
import {
  establishClientSession,
  isClientSessionEstablished,
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
});
