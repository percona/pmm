/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

import { describe, expect, it } from 'vitest';

import { getAtPath, setAtPath } from './fieldPath';

describe('fieldPath', () => {
  it('reads and writes nested dotted paths', () => {
    const values: Record<string, unknown> = {};
    setAtPath(values, 'source.mode', 'schema');
    setAtPath(values, 'source.source_db_id', 'inventory');

    expect(getAtPath(values, 'source.mode')).toBe('schema');
    expect(getAtPath(values, 'source.source_db_id')).toBe('inventory');
    expect(values).toEqual({
      source: { mode: 'schema', source_db_id: 'inventory' },
    });
  });

  it('reads nested values from caller default payloads', () => {
    const values = {
      source: { mode: 'query', source_query: 'SELECT 1' },
    };
    expect(getAtPath(values, 'source.source_query')).toBe('SELECT 1');
  });

  it('refuses prototype-pollution path segments on read', () => {
    const polluted = Object.create(null) as Record<string, unknown>;
    polluted.source = { mode: 'schema' };
    expect(getAtPath(polluted, '__proto__')).toBeUndefined();
    expect(getAtPath(polluted, 'constructor')).toBeUndefined();
    expect(getAtPath(polluted, 'source.__proto__.polluted')).toBeUndefined();
    expect(getAtPath(polluted, 'source.prototype.toString')).toBeUndefined();
  });

  it('refuses prototype-pollution path segments on write', () => {
    const values: Record<string, unknown> = {};
    setAtPath(values, '__proto__.polluted', true);
    setAtPath(values, 'source.constructor', 'bad');
    setAtPath(values, 'source.prototype.x', 1);
    expect(values).toEqual({});
    expect(
      (Object.prototype as { polluted?: boolean }).polluted
    ).toBeUndefined();
  });

  it('clones existing intermediate objects so writes do not mutate the source tree', () => {
    const nested = { mode: 'schema', source_db_id: 'inventory' };
    const values = { source: nested };
    const out: Record<string, unknown> = { ...values };

    setAtPath(out, 'source.mode', 'query');

    expect(nested.mode).toBe('schema');
    expect(nested.source_db_id).toBe('inventory');
    expect(out).toEqual({
      source: { mode: 'query', source_db_id: 'inventory' },
    });
    expect(out.source).not.toBe(nested);
  });

  it('clones intermediates along the path for deeper nested writes', () => {
    const inner = { leaf: 1 };
    const middle = { inner };
    const values = { outer: middle };

    const out: Record<string, unknown> = { ...values };
    setAtPath(out, 'outer.inner.leaf', 2);

    expect(inner.leaf).toBe(1);
    expect(getAtPath(out, 'outer.inner.leaf')).toBe(2);
    expect((out.outer as typeof middle).inner).not.toBe(inner);
    expect((out.outer as typeof middle).inner.leaf).toBe(2);
  });
});
