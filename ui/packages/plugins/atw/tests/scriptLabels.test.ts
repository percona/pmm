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
import { scriptLabel, scriptLabels } from '../src/scriptLabels';

describe('scriptLabel', () => {
  it('prefers the trimmed title', () => {
    expect(scriptLabel(' Summary ', 'a.sh')).toBe('Summary');
  });

  it('falls back to the filename for a missing or blank title', () => {
    expect(scriptLabel(undefined, 'a.sh')).toBe('a.sh');
    expect(scriptLabel(null, 'a.sh')).toBe('a.sh');
    expect(scriptLabel('  ', 'a.sh')).toBe('a.sh');
  });
});

describe('scriptLabels', () => {
  it('uses the title alone when no other file shares it', () => {
    const labels = scriptLabels([
      { title: 'Summary', filename: 'a.sh' },
      { title: 'Other', filename: 'b.sh' },
    ]);

    expect(labels.get('a.sh')).toBe('Summary');
    expect(labels.get('b.sh')).toBe('Other');
  });

  it('appends the filename when files share a title, ignoring surrounding whitespace', () => {
    const labels = scriptLabels([
      { title: 'Summary', filename: 'a.sh' },
      { title: 'Summary ', filename: 'b.sh' },
    ]);

    expect(labels.get('a.sh')).toBe('Summary (a.sh)');
    expect(labels.get('b.sh')).toBe('Summary (b.sh)');
  });

  it('does not treat two runs of one file as a collision', () => {
    const labels = scriptLabels([
      { title: 'Summary', filename: 'a.sh' },
      { title: 'Summary', filename: 'a.sh' },
    ]);

    expect(labels.get('a.sh')).toBe('Summary');
  });

  it('names an untitled file by its filename', () => {
    expect(scriptLabels([{ title: null, filename: 'a.sh' }]).get('a.sh')).toBe(
      'a.sh'
    );
  });
});
