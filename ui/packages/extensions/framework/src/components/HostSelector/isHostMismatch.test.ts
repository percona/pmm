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
import { isHostMismatch } from './isHostMismatch';

describe('isHostMismatch', () => {
  it('is silent when the host address is missing', () => {
    expect(isHostMismatch(undefined, '10.0.0.1')).toBe(false);
  });

  it('is silent when the host address is empty', () => {
    expect(isHostMismatch('', '10.0.0.1')).toBe(false);
  });

  it('is silent when the service node address is missing', () => {
    expect(isHostMismatch('10.0.0.1', undefined)).toBe(false);
  });

  it('is silent when the service node address is empty', () => {
    expect(isHostMismatch('10.0.0.1', '')).toBe(false);
  });

  it('is silent when both addresses are missing', () => {
    expect(isHostMismatch(undefined, undefined)).toBe(false);
  });

  it('is silent when both addresses match', () => {
    // Two Nomad names (or a Nomad name vs inventory display name) that
    // share one address are the same machine — names are not compared.
    expect(isHostMismatch('10.0.0.1', '10.0.0.1')).toBe(false);
  });

  it('warns only when the addresses differ', () => {
    expect(isHostMismatch('10.0.0.1', '10.0.0.2')).toBe(true);
  });
});
