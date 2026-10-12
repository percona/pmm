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
 * Hold every Operations surface to OmError's rule. This fails on the ways that bypass
 * the component: an error Alert of its own, and text coloured as an error.
 */

import { readdirSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const SRC = path.join(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
  'src'
);

/** The component that is allowed to draw an error frame, because it is the frame. */
const OWN = path.join(SRC, 'components', 'OmError.tsx');

const ERROR_ALERT = /severity="error"/;
const ERROR_TEXT =
  /<Typography\b[^>]*\scolor="error"|['"]error\.(?:main|light|dark)['"]/g;

/**
 * Red that marks a state, not a message: a count to act on, or a service or node that
 * is down or failing, its error a hover or an expand away. Counted exactly, so a red
 * message added to one of these files still fails.
 */
const STATUS_RED: Record<string, number> = {
  // "N down"
  'FleetClustersTab.tsx': 2,
  // "N down", a service's "failing 3h"
  'FleetServicesTab.tsx': 2,
  // "N cannot be scanned", a failing node's row and the heading of its detail
  'NodesPage.tsx': 3,
  // a service that did not answer a scan
  'components/RunEntities.tsx': 1,
  // a failed install's summary line
  'components/RunProgress.tsx': 1,
};

function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const full = path.join(dir, entry);
    if (statSync(full).isDirectory()) {
      return sources(full);
    }
    return full.endsWith('.tsx') && full !== OWN ? [full] : [];
  });
}

describe('the Operations error idiom', () => {
  it('draws no error Alert outside OmError', () => {
    const offenders = sources(SRC).filter((file) =>
      ERROR_ALERT.test(readFileSync(file, 'utf8'))
    );
    expect(offenders.map((file) => path.relative(SRC, file))).toEqual([]);
  });

  it('colours text as an error only to mark a status', () => {
    const red = Object.fromEntries(
      sources(SRC).flatMap((file) => {
        const count = readFileSync(file, 'utf8').match(ERROR_TEXT)?.length ?? 0;
        return count ? [[path.relative(SRC, file), count]] : [];
      })
    );
    expect(red).toEqual(STATUS_RED);
  });
});
