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

import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useCopyToClipboard } from './useCopyToClipboard';

describe('useCopyToClipboard', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('reports the newest attempt when an older one settles last', async () => {
    const settle: Array<(error?: Error) => void> = [];
    vi.stubGlobal('isSecureContext', true);
    vi.stubGlobal('navigator', {
      clipboard: {
        writeText: () =>
          new Promise<void>((resolve, reject) => {
            settle.push((error) => (error ? reject(error) : resolve()));
          }),
      },
    });

    const { result } = renderHook(() => useCopyToClipboard());

    let first!: Promise<boolean>;
    let second!: Promise<boolean>;
    act(() => {
      first = result.current.copy('first');
      second = result.current.copy('second');
    });

    await act(async () => {
      settle[1]();
      await second;
    });
    expect(result.current.copied).toBe(true);

    await act(async () => {
      settle[0](new Error('denied'));
      await first;
    });
    expect(result.current.copied).toBe(true);
    expect(result.current.failed).toBe(false);
  });
});
