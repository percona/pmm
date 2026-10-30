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

/** Display names keyed by filename, appending the filename to a title that several files share. */
export function scriptLabels(
  scripts: readonly { title?: string | null; filename: string }[]
): Map<string, string> {
  const filenamesByTitle = new Map<string, Set<string>>();
  for (const { title, filename } of scripts) {
    const trimmed = title?.trim();
    if (trimmed) {
      filenamesByTitle.set(
        trimmed,
        (filenamesByTitle.get(trimmed) ?? new Set<string>()).add(filename)
      );
    }
  }
  const labels = new Map<string, string>();
  for (const { title, filename } of scripts) {
    const label = scriptLabel(title, filename);
    const shared = (filenamesByTitle.get(label)?.size ?? 0) > 1;
    labels.set(filename, shared ? `${label} (${filename})` : label);
  }
  return labels;
}
