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
  isHydratedReferenceOption,
  resolveReferenceOption,
} from './referenceOption';

describe('referenceOption', () => {
  const options = [
    { id: 7, name: 'mysql-prod', type: 'mysql' },
    { id: 9, name: 'pg-prod', type: 'postgresql' },
  ];

  it('detects hydrated option objects', () => {
    expect(isHydratedReferenceOption({ id: 7, name: 'mysql-prod' })).toBe(true);
    expect(isHydratedReferenceOption(7)).toBe(false);
    expect(isHydratedReferenceOption({ id: 7 })).toBe(false);
  });

  it('resolves scalar ids to inventory options', () => {
    expect(resolveReferenceOption(7, options)).toEqual(options[0]);
    expect(resolveReferenceOption('9', options)).toEqual(options[1]);
  });

  it('returns null for unknown ids until options include a match', () => {
    expect(resolveReferenceOption(99, options)).toBeNull();
  });
});
