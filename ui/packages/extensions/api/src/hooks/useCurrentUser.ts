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
import type { ApiError } from '../errors';
import type { components } from '../generated/main';
import { mainApi, throwOnApiError } from '../typed-client';

/**
 * Sample typed hook demonstrating the full codegen → openapi-fetch → React Query
 * pattern. `data` is typed as `CasdoorUser` directly from the generated spec;
 * `error` is an `ApiError` (thanks to `throwOnApiError`).
 *
 * Reference for new hooks: copy this structure when wrapping typed endpoints.
 */
type CurrentUser = components['schemas']['CasdoorUser'];

export function useCurrentUser() {
  return useQuery<CurrentUser, ApiError>({
    queryKey: ['users', 'me'],
    queryFn: () => throwOnApiError(mainApi.GET('/api/users/me')),
  });
}
