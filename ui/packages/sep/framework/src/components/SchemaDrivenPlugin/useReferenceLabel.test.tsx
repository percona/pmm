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

import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@sep/api';
import { QueryWrapper } from '../../../tests/queryWrapper';
import { useReferenceLabel } from './useReferenceLabel';
import type { SettingReference } from './taskConfiguration';

function labelOf(reference: SettingReference, value: unknown) {
  return renderHook(() => useReferenceLabel(reference, value), {
    wrapper: QueryWrapper,
  });
}

describe('useReferenceLabel', () => {
  afterEach(() => vi.restoreAllMocks());

  it('names a table under the schema it belongs to', async () => {
    const get = vi
      .spyOn(apiClient, 'get')
      .mockResolvedValue({ data: [{ id: 9, name: 'actor' }] });

    const { result } = labelOf({ kind: 'table', schemaId: 5 }, 9);

    await waitFor(() => expect(result.current).toBe('actor'));
    expect(get).toHaveBeenCalledWith('/sep/schemas/5/tables');
  });

  it('says it is loading rather than showing the id meanwhile', () => {
    vi.spyOn(apiClient, 'get').mockReturnValue(new Promise(() => {}));

    const { result } = labelOf({ kind: 'service', serviceTypes: ['mysql'] }, 1);

    expect(result.current).toBe('Loading…');
  });

  it('keeps an unplaced host under its node name', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue({
      data: [{ id: 'node-1', name: 'db-1', address: '10.0.0.1' }],
    });

    const { result } = labelOf({ kind: 'host' }, 'node-2');

    await waitFor(() => expect(result.current).not.toBe('Loading…'));
    expect(result.current).toBe('node-2');
  });

  it('looks nothing up for a value typed in by hand', () => {
    const get = vi.spyOn(apiClient, 'get');

    const { result } = labelOf({ kind: 'schema', serviceId: 1 }, 'staging');

    expect(result.current).toBe('staging');
    expect(get).not.toHaveBeenCalled();
  });
});
