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

/** Shape returned by `GET /api/config/alerts`. */
export interface AlertConfig {
  available: boolean;
}

export const ALERT_CONFIG_QUERY_KEY = ['config', 'alerts'] as const;

/**
 * Fetches whether at least one alert provider is configured on the backend
 * (`bool(alert_settings.PROVIDERS)`). Used by `<AlertOnFailField>` to decide
 * if the *Alert on failure* checkbox should be enabled.
 *
 * The result rarely changes within a session, so it's cached for 5 minutes.
 */
export function useAlertConfig() {
  return useQuery<AlertConfig>({
    queryKey: ALERT_CONFIG_QUERY_KEY,
    queryFn: async () => {
      const { data } = await apiClient.get<AlertConfig>('/config/alerts');
      return data;
    },
    staleTime: 5 * 60 * 1000,
  });
}
