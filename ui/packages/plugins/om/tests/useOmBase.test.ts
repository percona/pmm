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

import { describe, expect, it } from 'vitest';
import {
  OM_ROUTE_HOSTS,
  OM_ROUTE_INVENTORY,
  OM_ROUTE_SERVICES,
} from '../src/constants';
import { omBase } from '../src/useOmBase';

const MOUNT = '/sep/om';

describe('omBase', () => {
  it('is the path itself on the index route', () => {
    expect(omBase(MOUNT)).toBe(MOUNT);
  });

  it('ignores a trailing slash', () => {
    expect(omBase(`${MOUNT}/`)).toBe(MOUNT);
  });

  // The regression this exists for: with a relative `to=""` a tab resolved against
  // the current child route and clicking it did nothing, so the only way back was
  // the sidebar. These are the routes OmApp actually declares -- the cases here used
  // to name `topology`, `runs` and `clusters/:id`, which no longer exist, so the suite
  // passed while the patterns matched nothing that ships.
  it.each([OM_ROUTE_SERVICES, OM_ROUTE_HOSTS, OM_ROUTE_INVENTORY])(
    'strips the %s segment so an absolute link resolves to the mount',
    (route) => {
      expect(omBase(`${MOUNT}/${route}`)).toBe(MOUNT);
    }
  );

  it('does not care where the shell mounts the plugin', () => {
    expect(omBase(`/somewhere/else/${OM_ROUTE_HOSTS}`)).toBe('/somewhere/else');
  });

  // Only the declared routes are stripped, so an unknown segment is left whole rather
  // than half-stripped.
  it('leaves a path that matches no child route alone', () => {
    expect(omBase(`${MOUNT}/clusters`)).toBe(`${MOUNT}/clusters`);
  });
});
