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
 * React Query hooks for the runtime app enable/disable admin API (SEP-982).
 *
 * The admin listing (`GET /api/admin/apps/`) and the public navigation listing
 * (`GET /api/apps/`, see {@link useEnabledApps}) are distinct queries with
 * different shapes, so both are invalidated on a successful transition to keep
 * the management page, the sidebar, and the disabled-app route guard consistent.
 *
 * Types here are hand-written (no OpenAPI codegen) to mirror the sibling
 * `useEnabledApps` hook; keep them in sync with the backend `AppInfoResponse` /
 * `AppStateResponse` models if those evolve.
 */
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
} from '@tanstack/react-query';

import { apiClient } from '../client';
import type { ApiError } from '../errors';
import { ENABLED_APPS_QUERY_KEY } from './useEnabledApps';

/** The four-state app lifecycle. `ENABLING`/`DISABLING` are transitional. */
export type AppLifecycleState =
  | 'ENABLED'
  | 'DISABLED'
  | 'ENABLING'
  | 'DISABLING';

/** The transitional states a toggle may request; the backend rejects the
 *  terminal `ENABLED`/`DISABLED` edges with 409. */
export type TransitionalState = Extract<
  AppLifecycleState,
  'ENABLING' | 'DISABLING'
>;

/** Per-app entry returned by the admin `GET /api/admin/apps/` endpoint. */
export interface AdminApp {
  app_key: string;
  name: string;
  enabled: boolean;
  lifecycle_state: AppLifecycleState;
  toggleable: boolean;
  uri_path: string;
  css_class: string;
  sidebar: boolean;
  has_api_router: boolean;
}

/** Result of a state transition (`PUT .../state`, `POST .../force-disable`). */
export interface AppStateResult {
  app_key: string;
  enabled: boolean;
  lifecycle_state: AppLifecycleState;
}

export const ADMIN_APPS_QUERY_KEY = ['admin', 'apps'] as const;

/**
 * Shared `mutationKey` for every app-transition mutation. Lets the management
 * page scope its `useMutationState` row-lock lookup to these mutations alone,
 * rather than any pending mutation that happens to carry an `appKey` variable.
 */
export const ADMIN_APP_MUTATION_KEY = ['admin', 'apps', 'transition'] as const;

/** Base path (relative to the axios client's `/api` baseURL). */
const ADMIN_APPS_BASE = '/admin/apps';

/** Poll cadence used while any app is mid-transition. */
const TRANSITION_POLL_MS = 3000;

/** True while an app sits in a transitional (in-progress) lifecycle state. */
export function isTransitional(state: AppLifecycleState): boolean {
  return state === 'ENABLING' || state === 'DISABLING';
}

/**
 * Fetch every configured app with its lifecycle state for the management page.
 *
 * While any app is transitional the query polls every {@link TRANSITION_POLL_MS}
 * so a cooperative drain that completes server-side is reflected without a
 * manual reload; polling stops once every app has settled into a terminal
 * state. Pass `enabled: false` to skip fetching entirely (e.g. non-admins).
 */
export function useAdminApps(options: { enabled?: boolean } = {}) {
  return useQuery<AdminApp[], ApiError>({
    queryKey: ADMIN_APPS_QUERY_KEY,
    enabled: options.enabled ?? true,
    queryFn: async () => {
      const { data } = await apiClient.get<AdminApp[]>(`${ADMIN_APPS_BASE}/`);
      return data;
    },
    // Keep the last good listing if a single poll errors mid-drain, so the
    // refetchInterval keeps evaluating and the page does not freeze.
    placeholderData: keepPreviousData,
    refetchInterval: (query) => {
      const apps = query.state.data;
      return apps?.some((app) => isTransitional(app.lifecycle_state))
        ? TRANSITION_POLL_MS
        : false;
    },
  });
}

/**
 * Apply a transition result to the cached admin listing immediately, then
 * invalidate both apps queries so the management page reflects the new state
 * without waiting for the next poll, and the sidebar / route guard re-filter.
 */
function applyTransition(
  queryClient: QueryClient,
  result: AppStateResult
): void {
  queryClient.setQueryData<AdminApp[]>(ADMIN_APPS_QUERY_KEY, (apps) =>
    apps?.map((app) =>
      app.app_key === result.app_key
        ? {
            ...app,
            enabled: result.enabled,
            lifecycle_state: result.lifecycle_state,
          }
        : app
    )
  );
  queryClient.invalidateQueries({ queryKey: ADMIN_APPS_QUERY_KEY });
  queryClient.invalidateQueries({ queryKey: ENABLED_APPS_QUERY_KEY });
}

export interface SetAppStateVars {
  appKey: string;
  /** Must be a transitional state; the backend rejects terminal edges with 409. */
  lifecycleState: TransitionalState;
}

/**
 * Initiate a lifecycle transition via `PUT /api/admin/apps/{app_key}/state`.
 *
 * Enabling a disabled app sends `ENABLING`; disabling an enabled app sends
 * `DISABLING`. On success both the admin and public apps queries are
 * invalidated. The rejected {@link ApiError} carries the backend `detail` for
 * 404 (unknown key) / 409 (illegal transition or protected app).
 */
export function useSetAppState() {
  const queryClient = useQueryClient();
  return useMutation<AppStateResult, ApiError, SetAppStateVars>({
    mutationKey: ADMIN_APP_MUTATION_KEY,
    mutationFn: async ({ appKey, lifecycleState }) => {
      const { data } = await apiClient.put<AppStateResult>(
        `${ADMIN_APPS_BASE}/${encodeURIComponent(appKey)}/state`,
        { lifecycle_state: lifecycleState }
      );
      return data;
    },
    onSuccess: (result) => applyTransition(queryClient, result),
  });
}

export interface ForceDisableAppVars {
  appKey: string;
}

/**
 * Force-finalise a stuck drain via `POST /api/admin/apps/{app_key}/force-disable`.
 *
 * Only valid while the app is `DISABLING`; the backend returns 409 otherwise.
 * Invalidates both apps queries on success.
 */
export function useForceDisableApp() {
  const queryClient = useQueryClient();
  return useMutation<AppStateResult, ApiError, ForceDisableAppVars>({
    mutationKey: ADMIN_APP_MUTATION_KEY,
    mutationFn: async ({ appKey }) => {
      const { data } = await apiClient.post<AppStateResult>(
        `${ADMIN_APPS_BASE}/${encodeURIComponent(appKey)}/force-disable`
      );
      return data;
    },
    onSuccess: (result) => applyTransition(queryClient, result),
  });
}

/**
 * Map an {@link ApiError} from the admin apps endpoints to a readable message.
 *
 * 404/409 responses carry a `{ detail }` body which is preferred verbatim; 401
 * indicates a missing bearer token on a mutation (CSRF defense). Returns `null`
 * for no error so callers can use it directly as a render guard.
 */
export function appStateErrorMessage(
  error: ApiError | null | undefined
): string | null {
  if (!error) {
    return null;
  }
  const detail = (error.data as { detail?: unknown } | undefined)?.detail;
  const detailText = typeof detail === 'string' ? detail : null;
  switch (error.status) {
    case 401:
      return 'Your session is missing credentials for this action. Try signing in again.';
    case 403:
      return detailText ?? 'You need administrator access to manage apps.';
    case 404:
      return detailText ?? 'That app no longer exists.';
    case 409:
      return detailText ?? 'That state change is not allowed right now.';
    default:
      return detailText ?? error.message ?? 'Something went wrong.';
  }
}
