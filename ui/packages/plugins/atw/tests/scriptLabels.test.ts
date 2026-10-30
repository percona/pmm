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
import { scriptLabel, scriptLabeler } from '../src/scriptLabels';

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

describe('scriptLabeler', () => {
  it('uses the title alone when no other file shares it', () => {
    const label = scriptLabeler([
      { title: 'Summary', filename: 'a.sh' },
      { title: 'Other', filename: 'b.sh' },
    ]);

    expect(label('Summary', 'a.sh')).toBe('Summary');
    expect(label('Other', 'b.sh')).toBe('Other');
  });

  it('appends the filename when files share a title, ignoring surrounding whitespace', () => {
    const label = scriptLabeler([
      { title: 'Summary', filename: 'a.sh' },
      { title: 'Summary ', filename: 'b.sh' },
    ]);

    expect(label('Summary', 'a.sh')).toBe('Summary (a.sh)');
    expect(label('Summary ', 'b.sh')).toBe('Summary (b.sh)');
  });

  it('does not treat two runs of one file as a collision', () => {
    const label = scriptLabeler([
      { title: 'Summary', filename: 'a.sh' },
      { title: 'Summary', filename: 'a.sh' },
    ]);

    expect(label('Summary', 'a.sh')).toBe('Summary');
  });

  it('labels each entry of one file by its own title', () => {
    const label = scriptLabeler([
      { title: null, filename: 'a.sh' },
      { title: 'Summary', filename: 'a.sh' },
    ]);

    expect(label(null, 'a.sh')).toBe('a.sh');
    expect(label('Summary', 'a.sh')).toBe('Summary');
  });

  it('appends the filename when a fallback name matches another title', () => {
    const label = scriptLabeler([
      { title: null, filename: 'Summary' },
      { title: 'Summary', filename: 'b.sh' },
    ]);

    expect(label(null, 'Summary')).toBe('Summary (Summary)');
    expect(label('Summary', 'b.sh')).toBe('Summary (b.sh)');
  });
});
