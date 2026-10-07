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

import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import { buildValidationRules } from '../src/components/SchemaFormRenderer/utils/validationMapper';

/**
 * The whitespace-only validation message is selected by matching the pattern a
 * backend field publishes, and that pattern is declared in Python. Nothing the
 * compiler sees connects the two: if the backend pattern changes, the rule
 * silently falls back to the generic format message and the user loses the
 * message that tells them what to fix.
 *
 * The committed spec is the seam both sides meet at — a Python test keeps it
 * fresh against the live backend, so reading it here turns the drift into a
 * failing test on whichever side moves first.
 */
const SPEC_PATH = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
  '..',
  'api',
  'specs',
  'extensions.json'
);

/** Return every pattern the spec publishes on a shared field-definition property. */
function publishedPatterns(): string[] {
  const spec = JSON.parse(readFileSync(SPEC_PATH, 'utf8')) as {
    components: {
      schemas: Record<string, { properties?: Record<string, unknown> }>;
    };
  };
  const found = new Set<string>();
  for (const schema of Object.values(spec.components.schemas)) {
    const destructive = schema.properties?.destructive as
      | { pattern?: string; anyOf?: { pattern?: string }[] }
      | undefined;
    if (!destructive) {
      continue;
    }
    for (const member of [destructive, ...(destructive.anyOf ?? [])]) {
      if (typeof member.pattern === 'string') {
        found.add(member.pattern);
      }
    }
  }
  return [...found];
}

describe('non-whitespace pattern — backend/frontend agreement', () => {
  it('names the whitespace-only case for the pattern the backend publishes', () => {
    const patterns = publishedPatterns();

    expect(patterns).toHaveLength(1);
    const rules = buildValidationRules({
      type: 'string',
      name: 'backup_dir',
      label: 'Backup directory',
      pattern: patterns[0],
    });
    expect((rules.pattern as { message: string }).message).toBe(
      'Backup directory cannot be only whitespace'
    );
  });
});
