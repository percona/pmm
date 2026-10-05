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

import { Navigate } from 'react-router-dom';
import { useOmBase } from './useOmBase';

/**
 * Where a retired route lands, resolved against the mount path.
 *
 * `to` is a value of `OM_LEGACY_REDIRECTS`: a child route, optionally with a
 * query string, and possibly *only* a query string, since the fleet is the index route
 * and its constant is `''`. Split rather than concatenated, so `services` lands on
 * `/operations?tab=services` and not on `/operations/?tab=services`.
 */
export function legacyRedirectTarget(
  base: string,
  to: string
): { pathname: string; search: string } {
  const [path, query] = to.split('?');
  return {
    pathname: path ? `${base}/${path}` : base || '/',
    search: query ? `?${query}` : '',
  };
}

/**
 * A retired route's redirect. Absolute against the mount, like every other link in the
 * plugin -- a bare `/${to}` resolved from the router root and left the plugin entirely,
 * sending `hosts` to `/nodes` and `services` to the root.
 */
export const LegacyRedirect = ({ to }: { to: string }) => {
  const base = useOmBase();
  return <Navigate to={legacyRedirectTarget(base, to)} replace />;
};
