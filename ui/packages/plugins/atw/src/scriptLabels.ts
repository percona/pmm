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

/** A script's display name: its title, or its filename when it has none. */
export function scriptLabel(
  title: string | null | undefined,
  filename: string
): string {
  return title?.trim() || filename;
}

/** Labels a script by its own title, appending its filename when another file in `scripts` resolves to the same label. */
export function scriptLabeler(
  scripts: readonly { title?: string | null; filename: string }[]
): (title: string | null | undefined, filename: string) => string {
  const filenamesByLabel = new Map<string, Set<string>>();
  for (const { title, filename } of scripts) {
    const label = scriptLabel(title, filename);
    filenamesByLabel.set(
      label,
      (filenamesByLabel.get(label) ?? new Set<string>()).add(filename)
    );
  }
  return (title, filename) => {
    const label = scriptLabel(title, filename);
    const filenames = filenamesByLabel.get(label) ?? new Set<string>();
    const shared = [...filenames].some((other) => other !== filename);
    return shared ? `${label} (${filename})` : label;
  };
}
