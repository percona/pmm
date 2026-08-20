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
import { omBase } from '../src/useOmBase';

const MOUNT = '/sep/om';

describe('omBase', () => {
  it('is the path itself on the index route', () => {
    expect(omBase(MOUNT)).toBe(MOUNT);
  });

  it('ignores a trailing slash', () => {
    expect(omBase(`${MOUNT}/`)).toBe(MOUNT);
  });

  // The regression: with a relative `to=""` the Clusters tab resolved to /runs
  // and clicking it did nothing, so the only way back was the sidebar.
  it('strips the runs segment so the Clusters tab can link back', () => {
    expect(omBase(`${MOUNT}/runs`)).toBe(MOUNT);
  });

  it('strips the topology segment so the Overview link resolves to the mount', () => {
    expect(omBase(`${MOUNT}/topology`)).toBe(MOUNT);
  });

  it('strips a cluster detail segment so the breadcrumb links back', () => {
    expect(omBase(`${MOUNT}/clusters/cl_8087e8bb`)).toBe(MOUNT);
  });

  it('does not care where the shell mounts the plugin', () => {
    expect(omBase('/somewhere/else/runs')).toBe('/somewhere/else');
  });

  // `clusters` alone is not a route OM declares; only `clusters/:id` is, so a
  // bare segment must be left alone rather than half-stripped.
  it('leaves a path that matches no child route alone', () => {
    expect(omBase(`${MOUNT}/clusters`)).toBe(`${MOUNT}/clusters`);
  });
});
