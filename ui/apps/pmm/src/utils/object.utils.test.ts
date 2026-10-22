import { describe, expect, it } from 'vitest';
import { isPlainObject } from './object.utils';

describe('isPlainObject', () => {
  it('accepts object literals and null-prototype objects', () => {
    expect(isPlainObject({})).toBe(true);
    expect(isPlainObject({ a: 1 })).toBe(true);
    expect(isPlainObject(JSON.parse('{"a":1}'))).toBe(true);
    expect(isPlainObject(Object.create(null))).toBe(true);
  });

  it('rejects values whose data is not in own enumerable properties', () => {
    class Query {}

    expect(isPlainObject(new Date())).toBe(false);
    expect(isPlainObject(new Map([['a', 1]]))).toBe(false);
    expect(isPlainObject(new Set([1]))).toBe(false);
    expect(isPlainObject(new Query())).toBe(false);
    expect(isPlainObject([1, 2])).toBe(false);
  });

  it('rejects primitives and null', () => {
    expect(isPlainObject(null)).toBe(false);
    expect(isPlainObject(undefined)).toBe(false);
    expect(isPlainObject('a')).toBe(false);
    expect(isPlainObject(1)).toBe(false);
  });
});
