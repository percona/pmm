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
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { OM_LEGACY_REDIRECTS } from '../src/constants';
import { LegacyRedirect, legacyRedirectTarget } from '../src/LegacyRedirect';

const MOUNT = '/operations';

const Where = () => {
  const { pathname, search } = useLocation();
  return <div data-testid="where">{`${pathname}${search}`}</div>;
};

// Mounted as the shell mounts the plugin, a splat under a path of its own, because a
// redirect that ignores the mount only shows up when there is one.
const landsOn = (from: string) => {
  render(
    <MemoryRouter initialEntries={[`${MOUNT}/${from}`]}>
      <Routes>
        <Route
          path={`${MOUNT}/*`}
          element={
            <Routes>
              {Object.entries(OM_LEGACY_REDIRECTS).map(([path, to]) => (
                <Route
                  key={path}
                  path={path}
                  element={<LegacyRedirect to={to} />}
                />
              ))}
              <Route path="*" element={<Where />} />
            </Routes>
          }
        />
        <Route path="*" element={<div data-testid="where">outside</div>} />
      </Routes>
    </MemoryRouter>
  );
  return screen.getByTestId('where').textContent;
};

describe('LegacyRedirect', () => {
  it.each([
    ['services', `${MOUNT}?tab=services`],
    ['hosts', `${MOUNT}/nodes`],
    ['hosts/bootstrap', `${MOUNT}/nodes/install`],
    ['inventory', `${MOUNT}/automations?tab=scans`],
  ])('sends %s to %s, inside the mount', (from, expected) => {
    expect(landsOn(from)).toBe(expected);
  });

  it.each([
    ['hosts/bootstrap?hosts=abc,def', `${MOUNT}/nodes/install?nodes=abc%2Cdef`],
    [
      'inventory?tab=config&period=7d',
      `${MOUNT}/automations?tab=scans&period=7d`,
    ],
  ])('carries the query of %s through to %s', (from, expected) => {
    expect(landsOn(from)).toBe(expected);
  });

  it('covers every retired route', () => {
    expect(Object.keys(OM_LEGACY_REDIRECTS).sort()).toEqual(
      ['hosts', 'hosts/bootstrap', 'inventory', 'services'].sort()
    );
  });
});

describe('legacyRedirectTarget', () => {
  it('keeps a query-only target on the index route, with no trailing slash', () => {
    expect(legacyRedirectTarget(MOUNT, '?tab=services')).toEqual({
      pathname: MOUNT,
      search: '?tab=services',
    });
  });

  it('falls back to the root when mounted at the root', () => {
    expect(legacyRedirectTarget('', '?tab=services').pathname).toBe('/');
  });
});
