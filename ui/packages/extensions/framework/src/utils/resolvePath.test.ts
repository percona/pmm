/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

import { describe, expect, it } from 'vitest';

import { resolvePath } from './resolvePath';

describe('resolvePath', () => {
  it('resolves a nested dotted path', () => {
    expect(resolvePath({ a: { b: { c: 1 } } }, 'a.b.c')).toBe(1);
  });

  it('returns undefined when an intermediate node is missing', () => {
    expect(resolvePath({ a: {} }, 'a.b.c')).toBeUndefined();
  });

  it('returns undefined when an intermediate node is a primitive', () => {
    expect(resolvePath({ a: 1 }, 'a.b')).toBeUndefined();
  });

  it('returns the verbatim primitive leaf, including falsy values', () => {
    expect(resolvePath({ a: 0 }, 'a')).toBe(0);
    expect(resolvePath({ a: false }, 'a')).toBe(false);
    expect(resolvePath({ a: '' }, 'a')).toBe('');
  });

  it('returns null leaves verbatim — caller applies its own empty guard', () => {
    expect(resolvePath({ a: null }, 'a')).toBeNull();
  });

  it('resolves array index syntax', () => {
    expect(resolvePath({ xs: [{ n: 1 }, { n: 2 }] }, 'xs[1].n')).toBe(2);
  });

  it('resolves multi-dimensional array indices', () => {
    expect(
      resolvePath(
        {
          m: [
            [1, 2],
            [3, 4],
          ],
        },
        'm[1][0]'
      )
    ).toBe(3);
  });

  it('returns undefined when an array index is out of bounds', () => {
    expect(resolvePath({ xs: [1] }, 'xs[5]')).toBeUndefined();
  });

  it('returns undefined for an empty path', () => {
    expect(resolvePath({ a: 1 }, '')).toBeUndefined();
  });

  it('returns undefined when record is null or undefined', () => {
    expect(resolvePath(null, 'a')).toBeUndefined();
    expect(resolvePath(undefined, 'a')).toBeUndefined();
  });

  it('returns undefined when an array index syntax targets a non-array', () => {
    expect(resolvePath({ xs: { 0: 'oops' } }, 'xs[0]')).toBeUndefined();
  });

  describe('security', () => {
    it('refuses to walk __proto__', () => {
      expect(resolvePath({}, '__proto__')).toBeUndefined();
    });

    it('refuses to walk constructor', () => {
      expect(resolvePath({}, 'constructor')).toBeUndefined();
    });

    it('refuses to walk prototype', () => {
      expect(resolvePath({ a: {} }, 'a.prototype.toString')).toBeUndefined();
    });

    it('refuses nested __proto__ traversal', () => {
      expect(resolvePath({ a: {} }, 'a.__proto__.polluted')).toBeUndefined();
    });

    it('refuses inherited properties — own-property-only', () => {
      const proto = { inherited: 1 };
      const child = Object.create(proto) as Record<string, unknown>;
      child.own = 2;
      expect(resolvePath(child, 'own')).toBe(2);
      expect(resolvePath(child, 'inherited')).toBeUndefined();
    });

    it('returns undefined for invalid segments', () => {
      expect(resolvePath({ 'foo-bar': 1 }, 'foo-bar')).toBeUndefined();
      expect(resolvePath({ '1foo': 1 }, '1foo')).toBeUndefined();
      expect(resolvePath({ a: 1 }, 'foo bar')).toBeUndefined();
    });
  });
});
