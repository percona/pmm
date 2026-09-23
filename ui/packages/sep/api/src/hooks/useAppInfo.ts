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

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../client';

/** Shape returned by `GET /api/sep/app-info`. */
export interface AppInfo {
  /** Rendered sidebar footer text (application summary and version by default). */
  footer_text: string;
}

export const APP_INFO_QUERY_KEY = ['sep', 'app-info'] as const;

/**
 * Fetches shell metadata for the sidebar footer (`footer_text`), mirroring the
 * value the legacy Jinja interface renders from `SEP__FOOTER_TEMPLATE`.
 *
 * The footer reflects a deployment-specific override but changes rarely within
 * a session, so the result is cached for 5 minutes.
 */
export function useAppInfo() {
  return useQuery<AppInfo>({
    queryKey: APP_INFO_QUERY_KEY,
    queryFn: async () => {
      const { data } = await apiClient.get<AppInfo>('/sep/app-info/');
      return data;
    },
    staleTime: 5 * 60 * 1000,
  });
}
