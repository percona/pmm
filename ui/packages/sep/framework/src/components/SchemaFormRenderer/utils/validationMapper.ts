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

import type { RegisterOptions } from 'react-hook-form';
import type { PluginField } from '../types';
import { getAtPath, setAtPath } from './fieldPath';

export function buildValidationRules(field: PluginField): RegisterOptions {
  const rules: RegisterOptions = {};

  if (field.required) {
    rules.required = `${field.label} is required`;
  }

  switch (field.type) {
    case 'string': {
      if (field.min_length !== undefined) {
        rules.minLength = {
          value: field.min_length,
          message: `Minimum ${field.min_length} characters`,
        };
      }
      if (field.max_length !== undefined) {
        rules.maxLength = {
          value: field.max_length,
          message: `Maximum ${field.max_length} characters`,
        };
      }
      if (field.pattern) {
        try {
          const compiled = new RegExp(field.pattern);
          rules.pattern = {
            value: compiled,
            message: `${field.label} does not match the required format`,
          };
        } catch {
          // Invalid regex from schema. Surface as a validate fn so the field
          // shows an error instead of the whole form crashing on render.
          rules.validate = () =>
            `${field.label} has an invalid pattern in its schema`;
        }
      }
      break;
    }
    case 'integer':
    case 'float': {
      // Reject non-numeric and (for integers) fractional input before
      // coercion: `coerceFormValues` used to truncate '2.5' to 2 and '3.14abc'
      // to 3.14, sending a value the user never typed.
      rules.validate = (value: unknown) => {
        if (value === '' || value === null || value === undefined) {
          return true;
        }
        const numericValue = Number(value);
        if (!Number.isFinite(numericValue)) {
          return 'Enter a valid number';
        }
        if (field.type === 'integer' && !Number.isInteger(numericValue)) {
          return 'Enter a whole number';
        }
        return true;
      };
      if (field.ge !== undefined) {
        rules.min = {
          value: field.ge,
          message: `Minimum value is ${field.ge}`,
        };
      }
      if (field.le !== undefined) {
        rules.max = {
          value: field.le,
          message: `Maximum value is ${field.le}`,
        };
      }
      break;
    }
    case 'textarea':
    case 'yaml': {
      if (field.type === 'textarea' || field.type === 'yaml') {
        // textarea/yaml share required-only rules today. Add length/pattern here if added later.
      }
      break;
    }
    case 'multi_choice': {
      const { min_items: minItems, max_items: maxItems, label } = field;
      if (minItems !== undefined || maxItems !== undefined) {
        rules.validate = (value: unknown) => {
          const len = Array.isArray(value) ? value.length : 0;
          if (minItems !== undefined && len < minItems) {
            return `Select at least ${minItems} ${label.toLowerCase()}`;
          }
          if (maxItems !== undefined && len > maxItems) {
            return `Select at most ${maxItems} ${label.toLowerCase()}`;
          }
          return true;
        };
      }
      break;
    }
    default:
      break;
  }

  return rules;
}

/**
 * Coerce raw form values into the types the backend expects.
 *
 * react-hook-form's Controller does not expose setValueAs, so integer/float
 * fields reach the submit handler as strings. This walks the schema and
 * converts values based on their declared type. Empty strings become
 * undefined so optional numeric fields serialise cleanly.
 */
export function coerceFormValues(
  values: Record<string, unknown>,
  fields: PluginField[]
): Record<string, unknown> {
  const out: Record<string, unknown> = { ...values };
  for (const field of fields) {
    const raw = getAtPath(out, field.name);
    if (field.type === 'integer' || field.type === 'float') {
      if (raw === '' || raw === undefined || raw === null) {
        setAtPath(out, field.name, undefined);
        continue;
      }
      // `Number` (not parseInt/parseFloat) so partly-numeric input stays raw
      // instead of being silently truncated; the field's validate rule rejects
      // it before submit.
      const num = Number(raw);
      const valid =
        Number.isFinite(num) &&
        (field.type !== 'integer' || Number.isInteger(num));
      setAtPath(out, field.name, valid ? num : raw);
      continue;
    }
    if (
      field.type === 'service' ||
      field.type === 'schema' ||
      field.type === 'table' ||
      field.type === 'host'
    ) {
      if (
        raw &&
        typeof raw === 'object' &&
        'id' in (raw as Record<string, unknown>)
      ) {
        const id = (raw as { id: unknown }).id;
        setAtPath(out, field.name, id ?? undefined);
      } else if (raw === '' || raw === null) {
        setAtPath(out, field.name, undefined);
      }
    }
  }
  return out;
}
