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

import { useLocation } from 'react-router-dom';

/** The child routes `PomApp` declares, as patterns to strip off the mount path. */
const CHILD_PATTERNS = [/\/topology$/, /\/runs$/, /\/clusters\/[^/]+$/];

/**
 * Recover POM's mount path from the current location.
 *
 * The shell mounts POM as a splat (`sep/pom/*`), so the plugin is never told
 * where it lives. Relative links are not a substitute: React Router resolves
 * `to=""` against the *current* route, so on `runs` it points back at `runs` —
 * which made the Clusters tab silently do nothing once you were on Discovery
 * runs. Deriving the base and linking absolutely is unambiguous from any route.
 *
 * Returns a path with no trailing slash, without the router basename — which is
 * exactly what `<Link to>` expects for an absolute target.
 */
export function pomBase(pathname: string): string {
  const trimmed = pathname.replace(/\/+$/, '');
  for (const pattern of CHILD_PATTERNS) {
    if (pattern.test(trimmed)) {
      return trimmed.replace(pattern, '');
    }
  }
  return trimmed;
}

/** `pomBase` bound to the current location. */
export function usePomBase(): string {
  return pomBase(useLocation().pathname);
}
