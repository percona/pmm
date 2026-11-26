/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

import { describe, expect, it } from 'vitest';

import { evaluatePredicate, getGateFieldNames } from './predicateEvaluator';

describe('evaluatePredicate – contains', () => {
  it('returns true when the value is in the array', () => {
    expect(
      evaluatePredicate(
        { contains: { upload: 's3' } },
        { upload: ['s3', 'rsync'] }
      )
    ).toBe(true);
  });

  it('returns false when the value is not in the array', () => {
    expect(
      evaluatePredicate({ contains: { upload: 's3' } }, { upload: ['rsync'] })
    ).toBe(false);
  });

  it('returns false when the container is undefined / null', () => {
    expect(evaluatePredicate({ contains: { upload: 's3' } }, {})).toBe(false);
    expect(
      evaluatePredicate({ contains: { upload: 's3' } }, { upload: null })
    ).toBe(false);
  });

  it('returns false when the container is empty', () => {
    expect(
      evaluatePredicate({ contains: { upload: 's3' } }, { upload: [] })
    ).toBe(false);
  });

  it('returns false when the container is not an array', () => {
    expect(
      evaluatePredicate({ contains: { upload: 's3' } }, { upload: 's3' })
    ).toBe(false);
    expect(evaluatePredicate({ contains: { upload: 1 } }, { upload: 42 })).toBe(
      false
    );
  });

  it('supports a $field reference on the right-hand side', () => {
    expect(
      evaluatePredicate(
        { contains: { upload: { $field: 'wanted' } } },
        { upload: ['s3', 'rsync'], wanted: 'rsync' }
      )
    ).toBe(true);
    expect(
      evaluatePredicate(
        { contains: { upload: { $field: 'wanted' } } },
        { upload: ['s3'], wanted: 'rsync' }
      )
    ).toBe(false);
  });

  it('contributes its field name to getGateFieldNames', () => {
    expect(
      getGateFieldNames([{ when: { contains: { upload: 's3' } } }])
    ).toEqual(['upload']);
  });
});

describe('evaluatePredicate – prototype-chain safety', () => {
  // Untrusted schema JSON could name a field `constructor`, `__proto__`, or
  // `toString`. Bracket access on a plain object resolves those to inherited
  // members on Object.prototype — `values.constructor` is the Object function
  // (truthy), `values.toString` is a function (truthy). The predicate must
  // treat absent own-properties as absent regardless of prototype chain.
  it('truthy: returns false for inherited members not present as own keys', () => {
    expect(evaluatePredicate({ truthy: 'constructor' }, {})).toBe(false);
    expect(evaluatePredicate({ truthy: 'toString' }, {})).toBe(false);
    expect(evaluatePredicate({ truthy: 'hasOwnProperty' }, {})).toBe(false);
    expect(evaluatePredicate({ truthy: '__proto__' }, {})).toBe(false);
  });

  it('falsy: returns true for inherited members not present as own keys', () => {
    expect(evaluatePredicate({ falsy: 'constructor' }, {})).toBe(true);
    expect(evaluatePredicate({ falsy: 'toString' }, {})).toBe(true);
  });

  it('all_present / none_present respect own-property semantics', () => {
    expect(evaluatePredicate({ all_present: ['constructor'] }, {})).toBe(false);
    expect(
      evaluatePredicate({ none_present: ['constructor', 'toString'] }, {})
    ).toBe(true);
  });

  it('still honours an own-property named like an inherited member', () => {
    // If the form genuinely binds a field called `constructor`, its own value
    // wins over the prototype chain.
    expect(
      evaluatePredicate({ truthy: 'constructor' }, { constructor: 'yes' })
    ).toBe(true);
    expect(
      evaluatePredicate({ truthy: 'constructor' }, { constructor: '' })
    ).toBe(false);
  });
});

describe('evaluatePredicate – $field reference to missing key', () => {
  it('equals: both sides absent collapses to true (undefined === undefined)', () => {
    // Documents existing behaviour — exposing it as an explicit case so
    // future changes have to consider it.
    expect(evaluatePredicate({ equals: { a: { $field: 'b' } } }, {})).toBe(
      true
    );
  });

  it('equals: only one side present is false', () => {
    expect(
      evaluatePredicate({ equals: { a: { $field: 'b' } } }, { a: 'x' })
    ).toBe(false);
  });

  it('contains: $field referencing missing key with present list is false', () => {
    // Container is array, needle is undefined — must not match.
    expect(
      evaluatePredicate(
        { contains: { upload: { $field: 'missing' } } },
        { upload: ['s3', 'rsync'] }
      )
    ).toBe(false);
  });
});

describe('evaluatePredicate – xor arity', () => {
  it('returns false for empty operand', () => {
    expect(evaluatePredicate({ xor: [] }, {})).toBe(false);
  });

  it('returns true for exactly one truthy child', () => {
    expect(
      evaluatePredicate(
        { xor: [{ truthy: 'a' }, { truthy: 'b' }] },
        { a: true, b: false }
      )
    ).toBe(true);
  });

  it('returns false when all children are truthy', () => {
    expect(
      evaluatePredicate(
        { xor: [{ truthy: 'a' }, { truthy: 'b' }] },
        { a: true, b: true }
      )
    ).toBe(false);
  });

  it('with 3 children uses exactly-one semantics', () => {
    expect(
      evaluatePredicate(
        { xor: [{ truthy: 'a' }, { truthy: 'b' }, { truthy: 'c' }] },
        { a: true, b: true, c: false }
      )
    ).toBe(false);
    expect(
      evaluatePredicate(
        { xor: [{ truthy: 'a' }, { truthy: 'b' }, { truthy: 'c' }] },
        { a: true, b: false, c: false }
      )
    ).toBe(true);
  });
});

describe('evaluatePredicate – empty / unknown / nested', () => {
  it('returns false for empty predicate object', () => {
    expect(evaluatePredicate({} as never, {})).toBe(false);
  });

  it('returns false for unknown operator', () => {
    expect(evaluatePredicate({ frobnicate: 'a' } as never, { a: true })).toBe(
      false
    );
  });

  it('all with empty operand is false (vacuous-truth guard)', () => {
    expect(evaluatePredicate({ all: [] }, {})).toBe(false);
  });

  it('all_truthy / all_falsy / all_present / none_present with empty operand are false', () => {
    expect(evaluatePredicate({ all_truthy: [] }, {})).toBe(false);
    expect(evaluatePredicate({ all_falsy: [] }, {})).toBe(false);
    expect(evaluatePredicate({ all_present: [] }, {})).toBe(false);
    expect(evaluatePredicate({ none_present: [] }, {})).toBe(false);
  });

  it('any with empty operand is false', () => {
    expect(evaluatePredicate({ any: [] }, {})).toBe(false);
  });

  it('three-level nested not/any/all evaluates correctly', () => {
    const pred = {
      not: {
        any: [{ all: [{ truthy: 'a' }, { truthy: 'b' }] }, { truthy: 'c' }],
      },
    };
    expect(evaluatePredicate(pred, { a: true, b: true, c: false })).toBe(false);
    expect(evaluatePredicate(pred, { a: true, b: false, c: false })).toBe(true);
    expect(evaluatePredicate(pred, { a: false, b: false, c: true })).toBe(
      false
    );
  });
});

describe('getGateFieldNames – nested predicates', () => {
  it('dedupes the same field referenced across nested branches', () => {
    expect(
      getGateFieldNames([
        {
          when: {
            any: [
              { truthy: 'upload' },
              { contains: { upload: 'S3' } },
              { all: [{ truthy: 'upload' }, { equals: { upload: 'S3' } }] },
            ],
          },
        },
      ])
    ).toEqual(['upload']);
  });

  it('collects $field reference targets in equals/contains', () => {
    expect(
      getGateFieldNames([{ when: { equals: { a: { $field: 'b' } } } }])
    ).toEqual(['a', 'b']);
  });
});

describe('evaluatePredicate – dotted field paths', () => {
  it('equals walks nested objects for dotted discriminator paths', () => {
    const values = {
      source: { mode: 'schema', source_db_id: 'inventory' },
    };
    expect(
      evaluatePredicate({ equals: { 'source.mode': 'schema' } }, values)
    ).toBe(true);
    expect(
      evaluatePredicate({ equals: { 'source.mode': 'query' } }, values)
    ).toBe(false);
  });

  it('not_equals forbids inactive one-of branch leaves via dotted paths', () => {
    const values = {
      source: { mode: 'schema', source_db_id: 'x', source_query: 'SELECT 1' },
    };
    expect(
      evaluatePredicate({ not_equals: { 'source.mode': 'query' } }, values)
    ).toBe(true);
  });

  it('truthy resolves nested segments without prototype-chain leakage', () => {
    expect(
      evaluatePredicate(
        { truthy: 'source.mode' },
        { source: { mode: 'schema' } }
      )
    ).toBe(true);
    expect(evaluatePredicate({ truthy: 'source.mode' }, {})).toBe(false);
  });

  it('prefers an own-property key when the full dotted path is stored flat', () => {
    expect(
      evaluatePredicate(
        { equals: { 'source.mode': 'flat' } },
        { 'source.mode': 'flat', source: { mode: 'nested' } }
      )
    ).toBe(true);
  });
});
