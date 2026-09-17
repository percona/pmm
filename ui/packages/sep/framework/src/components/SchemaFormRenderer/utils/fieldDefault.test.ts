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

import { describe, expect, it } from 'vitest';
import {
  emptyFieldValue,
  fieldDefault,
  fieldPlaceholder,
} from './fieldDefault';
import type { PluginField } from '../types';

describe('fieldDefault', () => {
  it('uses the schema default when set', () => {
    const field: PluginField = {
      type: 'string',
      name: 'path',
      label: 'Log file path',
      default: '/var/log/mysql/error.log',
      placeholder: 'ignored-when-default-set',
    };
    expect(fieldDefault(field)).toBe('/var/log/mysql/error.log');
  });

  it('promotes placeholder to the starting value when default is unset', () => {
    const field: PluginField = {
      type: 'string',
      name: 'path',
      label: 'Log file path',
      placeholder: '/var/log/mysql/error.log',
    };
    expect(fieldDefault(field)).toBe('/var/log/mysql/error.log');
  });

  it('seeds a required integer with ge, else 1', () => {
    expect(
      fieldDefault({
        type: 'integer',
        name: 'minutes',
        label: 'Minutes',
        required: true,
        ge: 5,
      })
    ).toBe(5);
    expect(
      fieldDefault({
        type: 'integer',
        name: 'minutes',
        label: 'Minutes',
        required: true,
      })
    ).toBe(1);
  });

  it('seeds a required float with ge, else 1', () => {
    expect(
      fieldDefault({
        type: 'float',
        name: 'rate',
        label: 'Rate',
        required: true,
        ge: 0.5,
      })
    ).toBe(0.5);
    expect(
      fieldDefault({
        type: 'float',
        name: 'rate',
        label: 'Rate',
        required: true,
      })
    ).toBe(1);
  });

  it('leaves optional numbers empty when unset', () => {
    expect(
      fieldDefault({
        type: 'integer',
        name: 'optional',
        label: 'Optional',
      })
    ).toBe(
      emptyFieldValue({ type: 'integer', name: 'optional', label: 'Optional' })
    );
  });

  it('keeps an explicit numeric default of 0', () => {
    expect(
      fieldDefault({
        type: 'integer',
        name: 'count',
        label: 'Count',
        required: true,
        default: 0,
      })
    ).toBe(0);
  });
});

describe('fieldPlaceholder', () => {
  it('hides the placeholder when it was promoted into the value', () => {
    expect(
      fieldPlaceholder({
        type: 'string',
        name: 'path',
        label: 'Log file path',
        placeholder: '/var/log/mysql/error.log',
      })
    ).toBeUndefined();
  });

  it('keeps the placeholder as a hint when a real default is set', () => {
    expect(
      fieldPlaceholder({
        type: 'string',
        name: 'path',
        label: 'Log file path',
        default: '/var/log/mysql/error.log',
        placeholder: '/path/to/log',
      })
    ).toBe('/path/to/log');
  });
});
