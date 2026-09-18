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

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import type { RelatedApp } from '@sep/api';
import {
  RelatedAppTabBar,
  resolveRelatedAppActiveSegment,
} from './RelatedAppTabBar';

const ROUTE_BASE = '/apps/mysql_backups';

const RELATED_APPS: RelatedApp[] = [
  {
    app_key: 'mysql_backups/restore',
    label: 'Restore',
    route_segment: 'restores',
  },
];

function renderTabBar(pathname: string, activeSegment?: string) {
  const segment =
    activeSegment ??
    resolveRelatedAppActiveSegment(pathname, ROUTE_BASE, RELATED_APPS);

  return render(
    <MemoryRouter initialEntries={[pathname]}>
      <RelatedAppTabBar
        parentLabel="MySQL Backups"
        routeBase={ROUTE_BASE}
        relatedApps={RELATED_APPS}
        activeSegment={segment}
      />
    </MemoryRouter>
  );
}

describe('resolveRelatedAppActiveSegment', () => {
  it('returns empty string on the parent list route', () => {
    expect(
      resolveRelatedAppActiveSegment(
        '/apps/mysql_backups',
        ROUTE_BASE,
        RELATED_APPS
      )
    ).toBe('');
  });

  it('returns empty string on a parent task detail route', () => {
    expect(
      resolveRelatedAppActiveSegment(
        '/apps/mysql_backups/task/backup-1/logs',
        ROUTE_BASE,
        RELATED_APPS
      )
    ).toBe('');
  });

  it('returns the related segment on the related list route', () => {
    expect(
      resolveRelatedAppActiveSegment(
        '/apps/mysql_backups/restores',
        ROUTE_BASE,
        RELATED_APPS
      )
    ).toBe('restores');
  });

  it('returns the related segment on a nested related detail route', () => {
    expect(
      resolveRelatedAppActiveSegment(
        '/apps/mysql_backups/restores/task/restore-1',
        ROUTE_BASE,
        RELATED_APPS
      )
    ).toBe('restores');
  });
});

describe('RelatedAppTabBar', () => {
  it('renders parent and related tab labels with navigation targets', () => {
    renderTabBar('/apps/mysql_backups');

    const backupsTab = screen.getByRole('tab', { name: 'MySQL Backups' });
    const restoreTab = screen.getByRole('tab', { name: 'Restore' });

    expect(backupsTab).toHaveAttribute('href', '/apps/mysql_backups');
    expect(restoreTab).toHaveAttribute('href', '/apps/mysql_backups/restores');
  });

  it('selects the parent tab on parent routes', () => {
    renderTabBar('/apps/mysql_backups/task/backup-1');

    expect(screen.getByRole('tab', { name: 'MySQL Backups' })).toHaveAttribute(
      'aria-selected',
      'true'
    );
    expect(screen.getByRole('tab', { name: 'Restore' })).toHaveAttribute(
      'aria-selected',
      'false'
    );
  });

  it('selects the related tab on related routes', () => {
    renderTabBar('/apps/mysql_backups/restores/new');

    expect(screen.getByRole('tab', { name: 'MySQL Backups' })).toHaveAttribute(
      'aria-selected',
      'false'
    );
    expect(screen.getByRole('tab', { name: 'Restore' })).toHaveAttribute(
      'aria-selected',
      'true'
    );
  });
});
