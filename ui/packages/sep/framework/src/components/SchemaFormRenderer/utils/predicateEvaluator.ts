/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import type { FieldGate, Predicate } from '../types';

type FormValues = Record<string, unknown>;

/**
 * Read a field by name from form values, ignoring inherited members.
 *
 * Supports dotted paths (for example ``source.mode``) by walking nested plain
 * objects when no own-property exists for the full path string. Field names
 * come from JSON schema and are not fully trusted: a schema that names a field
 * ``constructor`` / ``toString`` / ``__proto__`` would otherwise resolve via
 * the prototype chain to truthy built-ins (e.g. the ``Object`` function),
 * making gate predicates evaluate against engine internals instead of
 * user-supplied state. Always go through this helper.
 */
function readField(values: FormValues, name: string): unknown {
  if (Object.prototype.hasOwnProperty.call(values, name)) {
    return values[name];
  }
  if (!name.includes('.')) {
    return undefined;
  }
  const segments = name.split('.');
  let current: unknown = values;
  for (const segment of segments) {
    if (
      !isPlainObject(current) ||
      !Object.prototype.hasOwnProperty.call(current, segment)
    ) {
      return undefined;
    }
    current = current[segment];
  }
  return current;
}

function isPlainObject(v: unknown): v is Record<string, unknown> {
  if (typeof v !== 'object' || v === null || Array.isArray(v)) {
    return false;
  }
  const proto = Object.getPrototypeOf(v) as unknown;
  return proto === Object.prototype || proto === null;
}

function isTruthy(value: unknown): boolean {
  if (Array.isArray(value)) {
    return value.length > 0;
  }
  // mirror Python bool({}): empty plain objects are falsy
  if (isPlainObject(value)) {
    return Object.keys(value).length > 0;
  }
  return Boolean(value);
}

export function isPresent(value: unknown): boolean {
  if (Array.isArray(value)) {
    return value.length > 0;
  }
  // mirror Python _field_is_present: empty plain objects are absent
  if (isPlainObject(value)) {
    return Object.keys(value).length > 0;
  }
  // ``false`` is the unset bool default and counts as absent so a
  // BoolField's ``forbidden=`` gate only fires on an explicit ``true``
  // toggle. Matches the Python convention introduced for ``BoolField``
  // gating; ``0`` stays present because numeric zero is a real value.
  if (value === false) {
    return false;
  }
  return value !== null && value !== undefined && value !== '';
}

function isFieldRef(v: unknown): v is { $field: string } {
  return typeof v === 'object' && v !== null && '$field' in v;
}

function resolveValue(raw: unknown, values: FormValues): unknown {
  return isFieldRef(raw)
    ? readField(values, (raw as { $field: string }).$field)
    : raw;
}

/**
 * Coerces a value to number for numeric comparisons.
 * Intentional divergence from Python (which raises TypeError on "10" > 5):
 * RHF stores text-input values as strings until coerceFormValues runs at submit,
 * so predicate evaluation must handle string representations of numbers.
 * Used for ordered comparisons (gt/gte/lt/lte) and equality against numeric
 * literals, because the BE wire format serialises IntEnum and int values as
 * JSON numbers (e.g. { equals: { swap_drop: 1 } }).
 */
function toNum(v: unknown): number {
  return typeof v === 'number' ? v : Number(v);
}

/**
 * Evaluate a predicate wire-format object against the current form values.
 *
 * Mirrors the backend DSL operators from `app/sep/plugins/framework/rules.py`.
 * Each predicate is a single-key object `{ <operator>: <operand> }` — the BE
 * schema validator enforces this; multi-key predicates are not valid wire format
 * and only the first entry would be evaluated.
 * Unknown operators return false so malformed schemas degrade gracefully.
 */
export function evaluatePredicate(
  predicate: Predicate,
  values: FormValues
): boolean {
  const entries = Object.entries(predicate);
  if (entries.length === 0) {
    return false;
  }
  const [op, operand] = entries[0];

  switch (op) {
    case 'truthy':
      return isTruthy(readField(values, operand as string));
    case 'falsy':
      return !isTruthy(readField(values, operand as string));

    case 'equals': {
      const entry = Object.entries(operand as Record<string, unknown>)[0];
      if (!entry) {
        return false;
      }
      const [field, raw] = entry;
      const rhs = resolveValue(raw, values);
      // Coerce field value when the literal is numeric: BE emits int/IntEnum
      // values as JSON numbers, but RHF stores text inputs as strings.
      const fieldValue = readField(values, field);
      const lhs = typeof rhs === 'number' ? toNum(fieldValue) : fieldValue;
      return lhs === rhs;
    }
    case 'not_equals': {
      const entry = Object.entries(operand as Record<string, unknown>)[0];
      if (!entry) {
        return false;
      }
      const [field, raw] = entry;
      const rhs = resolveValue(raw, values);
      const fieldValue = readField(values, field);
      const lhs = typeof rhs === 'number' ? toNum(fieldValue) : fieldValue;
      return lhs !== rhs;
    }
    case 'gt': {
      const entry = Object.entries(operand as Record<string, unknown>)[0];
      if (!entry) {
        return false;
      }
      const [field, raw] = entry;
      return toNum(readField(values, field)) > toNum(resolveValue(raw, values));
    }
    case 'gte': {
      const entry = Object.entries(operand as Record<string, unknown>)[0];
      if (!entry) {
        return false;
      }
      const [field, raw] = entry;
      return (
        toNum(readField(values, field)) >= toNum(resolveValue(raw, values))
      );
    }
    case 'lt': {
      const entry = Object.entries(operand as Record<string, unknown>)[0];
      if (!entry) {
        return false;
      }
      const [field, raw] = entry;
      return toNum(readField(values, field)) < toNum(resolveValue(raw, values));
    }
    case 'lte': {
      const entry = Object.entries(operand as Record<string, unknown>)[0];
      if (!entry) {
        return false;
      }
      const [field, raw] = entry;
      return (
        toNum(readField(values, field)) <= toNum(resolveValue(raw, values))
      );
    }

    case 'contains': {
      const entry = Object.entries(operand as Record<string, unknown>)[0];
      if (!entry) {
        return false;
      }
      const [field, raw] = entry;
      const container = readField(values, field);
      if (!Array.isArray(container)) {
        return false;
      }
      const needle = resolveValue(raw, values);
      return container.includes(needle);
    }

    case 'all_truthy': {
      const fs = operand as string[];
      return fs.length > 0 && fs.every((f) => isTruthy(readField(values, f)));
    }
    case 'all_falsy': {
      const fs = operand as string[];
      return fs.length > 0 && fs.every((f) => !isTruthy(readField(values, f)));
    }
    case 'any_truthy':
      return (operand as string[]).some((f) => isTruthy(readField(values, f)));
    case 'any_falsy':
      return (operand as string[]).some((f) => !isTruthy(readField(values, f)));
    case 'any_present':
      return (operand as string[]).some((f) => isPresent(readField(values, f)));
    case 'all_present': {
      const fs = operand as string[];
      return fs.length > 0 && fs.every((f) => isPresent(readField(values, f)));
    }
    case 'none_present': {
      const fs = operand as string[];
      return fs.length > 0 && fs.every((f) => !isPresent(readField(values, f)));
    }
    case 'all_equal': {
      const fields = operand as string[];
      if (fields.length < 2) {
        return false;
      }
      const first = readField(values, fields[0]);
      return fields.slice(1).every((f) => readField(values, f) === first);
    }

    case 'all': {
      const ps = operand as Predicate[];
      return ps.length > 0 && ps.every((p) => evaluatePredicate(p, values));
    }
    case 'any':
      return (operand as Predicate[]).some((p) => evaluatePredicate(p, values));
    case 'xor': {
      // BE Xor is binary-only by design (rules.py always emits exactly 2 children).
      // The "exactly one truthy" check is equivalent to binary parity for arity 2.
      // For arity >= 3 these diverge: binary parity = odd count truthy, exactly-one
      // = count === 1. If BE ever extends xor to n-ary it would use parity semantics,
      // so a new `exactly_one_of` operator should be introduced instead of reusing xor.
      const preds = operand as Predicate[];
      if (preds.length === 0) {
        return false;
      }
      return preds.filter((p) => evaluatePredicate(p, values)).length === 1;
    }
    case 'not':
      return !evaluatePredicate(operand as Predicate, values);

    default:
      return false;
  }
}

function getPredicateFieldNames(predicate: Predicate): string[] {
  const entries = Object.entries(predicate);
  if (entries.length === 0) {
    return [];
  }
  const [op, operand] = entries[0];

  switch (op) {
    case 'truthy':
    case 'falsy':
      return [operand as string];
    case 'equals':
    case 'not_equals':
    case 'gt':
    case 'gte':
    case 'lt':
    case 'lte':
    case 'contains': {
      const entry = Object.entries(operand as Record<string, unknown>)[0];
      if (!entry) {
        return [];
      }
      const [field, raw] = entry;
      const names: string[] = [field];
      if (isFieldRef(raw)) {
        names.push((raw as { $field: string }).$field);
      }
      return names;
    }
    case 'all_truthy':
    case 'all_falsy':
    case 'any_truthy':
    case 'any_falsy':
    case 'any_present':
    case 'all_present':
    case 'none_present':
    case 'all_equal':
      return operand as string[];
    case 'all':
    case 'any':
    case 'xor':
      return (operand as Predicate[]).flatMap((p) => getPredicateFieldNames(p));
    case 'not':
      return getPredicateFieldNames(operand as Predicate);
    default:
      return [];
  }
}

/** Returns the deduplicated set of field names referenced by a list of gates. */
export function getGateFieldNames(gates: FieldGate[]): string[] {
  return [...new Set(gates.flatMap((g) => getPredicateFieldNames(g.when)))];
}
