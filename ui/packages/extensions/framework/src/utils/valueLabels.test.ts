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
import { applyValueLabel } from './valueLabels';

const LABELS = { M: 'Mydumper', X: 'XtraBackup' };

describe('applyValueLabel', () => {
  it('maps a value the schema labels', () => {
    expect(applyValueLabel('M', LABELS)).toBe('Mydumper');
  });

  it('passes through a value the map does not mention', () => {
    expect(applyValueLabel('B', LABELS)).toBe('B');
  });

  it('passes through when the schema declares no map', () => {
    expect(applyValueLabel('M', undefined)).toBe('M');
  });

  it('leaves null and undefined alone', () => {
    expect(applyValueLabel(null, LABELS)).toBeNull();
    expect(applyValueLabel(undefined, LABELS)).toBeUndefined();
  });

  it('keys on the value string form, so a number matches a string key', () => {
    expect(applyValueLabel(1, { '1': 'One' })).toBe('One');
  });

  it('does not read inherited properties off the label map', () => {
    // A bare index would resolve these through Object.prototype and hand React
    // a function or an object, which it cannot render.
    for (const value of ['constructor', 'toString', 'valueOf', '__proto__']) {
      expect(applyValueLabel(value, LABELS)).toBe(value);
    }
  });

  it('still honours a map that genuinely declares such a key', () => {
    expect(applyValueLabel('constructor', { constructor: 'Builder' })).toBe(
      'Builder'
    );
  });
});
