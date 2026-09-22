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
import { ApiError, apiClient } from '@sep/api';
import { QueryWrapper } from '../../tests/queryWrapper';
import { useTaskFileDownload } from './useTaskFileDownload';

vi.mock('../utils/downloadBlob', () => ({ downloadBlob: vi.fn() }));

describe('useTaskFileDownload', () => {
  afterEach(() => vi.restoreAllMocks());

  it("surfaces a refusal's own reason, not the bare status", async () => {
    // The request asks for a blob, so the 403's JSON body arrives as one and the
    // reason is unreadable until it is parsed.
    vi.spyOn(apiClient, 'get').mockRejectedValue(
      new ApiError({
        kind: 'http',
        status: 403,
        message: 'HTTP 403',
        data: new Blob(
          [
            JSON.stringify({
              detail: "You don't have permission to perform this action",
            }),
          ],
          { type: 'application/json' }
        ),
      })
    );

    const { result } = renderHook(() => useTaskFileDownload(), {
      wrapper: QueryWrapper,
    });
    result.current.mutate({
      taskHistoryId: 7,
      path: 'out/log.txt',
      isDir: false,
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe(
      "You don't have permission to perform this action"
    );
  });

  it('downloads the file on success', async () => {
    const { downloadBlob } = await import('../utils/downloadBlob');
    const blob = new Blob(['hello'], { type: 'text/plain' });
    vi.spyOn(apiClient, 'get').mockResolvedValue({ data: blob });

    const { result } = renderHook(() => useTaskFileDownload(), {
      wrapper: QueryWrapper,
    });
    result.current.mutate({
      taskHistoryId: 7,
      path: 'out/log.txt',
      isDir: false,
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(downloadBlob).toHaveBeenCalledWith(blob, 'log.txt');
  });

  it('suggests a tarball name for a directory', async () => {
    const { downloadBlob } = await import('../utils/downloadBlob');
    const blob = new Blob(['tar'], { type: 'application/gzip' });
    vi.spyOn(apiClient, 'get').mockResolvedValue({ data: blob });

    const { result } = renderHook(() => useTaskFileDownload(), {
      wrapper: QueryWrapper,
    });
    result.current.mutate({ taskHistoryId: 7, path: 'out/logs', isDir: true });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(downloadBlob).toHaveBeenCalledWith(blob, 'logs.tar.gz');
  });
});
