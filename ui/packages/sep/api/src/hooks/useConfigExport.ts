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

/**
 * React Query hook for the admin config YAML export (SEP-984).
 *
 * Fetches ``GET /api/sep/admin/settings/export`` through ``apiClient`` so the
 * Bearer interceptor attaches the in-memory access token, then triggers a
 * browser download from the response blob and server-provided filename.
 */
import { useMutation } from '@tanstack/react-query';

import { apiClient } from '../client';
import type { ApiError } from '../errors';

const CONFIG_EXPORT_PATH = '/sep/admin/settings/export';
const DEFAULT_FILENAME = 'sep-config.yaml';

function filenameFromContentDisposition(
  header: string | undefined
): string | null {
  const match = /filename="([^"]+)"/.exec(header ?? '');
  return match?.[1] ?? null;
}

/** Trigger a browser download for an in-memory blob (mirrors ``@sep/framework``). */
function triggerBlobDownload(data: Blob, filename: string): void {
  const url = URL.createObjectURL(data);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.style.display = 'none';
  document.body.appendChild(anchor);
  try {
    anchor.click();
  } finally {
    anchor.remove();
    setTimeout(() => URL.revokeObjectURL(url), 0);
  }
}

/**
 * Download the merged effective configuration as YAML.
 *
 * Admin-only on the backend; callers should gate the trigger button on
 * ``useAuth().isAdmin``.
 */
export function useConfigExport() {
  return useMutation<void, ApiError, void>({
    mutationFn: async () => {
      const response = await apiClient.get<Blob>(CONFIG_EXPORT_PATH, {
        responseType: 'blob',
      });
      const header = response.headers['content-disposition'];
      const filename =
        filenameFromContentDisposition(header) ?? DEFAULT_FILENAME;
      triggerBlobDownload(response.data, filename);
    },
  });
}
