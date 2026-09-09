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

import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import type { RelatedApp } from '@sep/api';
import { Link } from 'react-router-dom';

export interface RelatedAppTabBarProps {
  /** Human-readable label for the parent app's default surface. */
  parentLabel: string;
  /** Absolute route prefix for the parent plugin (for example `/apps/mysql_backups`). */
  routeBase: string;
  relatedApps: RelatedApp[];
  /**
   * Active related-app segment, or `''` when the parent surface is selected.
   * Derived from the current location via `resolveRelatedAppActiveSegment`.
   */
  activeSegment: string;
}

/**
 * Return which related-app `route_segment` is active for `pathname`.
 *
 * The parent surface is represented by `''` when the path is under
 * `routeBase` but not under any `{routeBase}/{route_segment}` prefix.
 */
export function resolveRelatedAppActiveSegment(
  pathname: string,
  routeBase: string,
  relatedApps: RelatedApp[]
): string {
  const normalizedBase = routeBase.replace(/\/+$/, '');
  for (const related of relatedApps) {
    const prefix = `${normalizedBase}/${related.route_segment}`;
    if (pathname === prefix || pathname.startsWith(`${prefix}/`)) {
      return related.route_segment;
    }
  }
  return '';
}

/** Sibling-app tab bar for schema-driven plugins that declare `related_apps`. */
export function RelatedAppTabBar({
  parentLabel,
  routeBase,
  relatedApps,
  activeSegment,
}: RelatedAppTabBarProps) {
  const normalizedBase = routeBase.replace(/\/+$/, '');

  return (
    <Tabs
      value={activeSegment}
      aria-label={`${parentLabel} and related apps`}
      sx={{ mb: 3, borderBottom: 1, borderColor: 'divider' }}
    >
      <Tab
        label={parentLabel}
        value=""
        component={Link}
        to={normalizedBase}
        replace
      />
      {relatedApps.map((related) => (
        <Tab
          key={related.app_key}
          label={related.label}
          value={related.route_segment}
          component={Link}
          to={`${normalizedBase}/${related.route_segment}`}
          replace
        />
      ))}
    </Tabs>
  );
}
