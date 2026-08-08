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

import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { apiClient, type ChoiceOption } from '@sep/api';
import { sepRetry } from './sepRetry';

export interface UseRemoteChoicesOptions {
  /** Fully-resolved path the options are fetched from, relative to `apiClient`'s `/api` base. */
  endpointUrl: string;
  /** Sibling field name driving the cascade; `null`/`undefined` means no cascade. */
  dependsOnName?: string | null;
  /** Current value of the cascade parent; the fetch waits until this is set. */
  dependsOnValue?: string | null;
  enabled?: boolean;
}

/**
 * Build the effective request URL: preserve any query string already on
 * `endpointUrl` and, when a cascade dependency is provided, append it as a
 * query parameter named after `dependsOnName` via `URLSearchParams` (so the
 * value is encoded and a pre-existing `?` is not duplicated). Returns
 * `endpointUrl` unchanged when no cascade value is present.
 */
function buildEffectiveUrl(
  endpointUrl: string,
  dependsOnName?: string | null,
  dependsOnValue?: string | null
): string {
  if (
    dependsOnName === null ||
    dependsOnName === undefined ||
    dependsOnValue === null ||
    dependsOnValue === undefined
  ) {
    return endpointUrl;
  }
  const queryStart = endpointUrl.indexOf('?');
  const path =
    queryStart === -1 ? endpointUrl : endpointUrl.slice(0, queryStart);
  const existingQuery =
    queryStart === -1 ? '' : endpointUrl.slice(queryStart + 1);
  const params = new URLSearchParams(existingQuery);
  params.set(dependsOnName, dependsOnValue);
  return `${path}?${params.toString()}`;
}

/**
 * Fetch `Choice`-compatible options for a `RemoteChoices` field from its
 * wire-declared endpoint.
 *
 * Mirrors the reference selector hooks (React Query + `apiClient` +
 * `staleTime` + `sepRetry`), but reads the URL from `endpointUrl` rather than a
 * hardcoded template. A cascading field (one with `dependsOnName`) skips the
 * fetch until its parent has a value; a non-cascading field always fetches. The
 * query key carries the full effective URL, so two fields that share an endpoint
 * and parent value but cascade from differently-named parents do not collide.
 */
export function useRemoteChoices(
  options: UseRemoteChoicesOptions
): UseQueryResult<ChoiceOption[], Error> {
  const {
    endpointUrl,
    dependsOnName,
    dependsOnValue,
    enabled = true,
  } = options;
  const effectiveUrl = buildEffectiveUrl(
    endpointUrl,
    dependsOnName,
    dependsOnValue
  );
  const cascades = dependsOnName !== null && dependsOnName !== undefined;
  return useQuery<ChoiceOption[], Error>({
    queryKey: ['remote-choices', effectiveUrl],
    enabled:
      enabled &&
      (!cascades || (dependsOnValue !== null && dependsOnValue !== undefined)),
    staleTime: 60_000,
    queryFn: async () => {
      const response = await apiClient.get<ChoiceOption[]>(effectiveUrl);
      return response.data;
    },
    retry: sepRetry,
  });
}
