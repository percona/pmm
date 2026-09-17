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
 * Slugify a service name for use in a task-name suggestion.
 *
 * Lowercases, replaces runs of non-alphanumeric characters with a single
 * hyphen, and trims leading/trailing hyphens. Empty / whitespace-only input
 * yields an empty string.
 */
export function slugifyServiceName(name: string | undefined | null): string {
  if (!name) {
    return '';
  }
  return name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

/**
 * Build a non-empty suggested MongoDB backup task name.
 *
 * Format: ``{serviceSlug|mongodb}-backup-{UTC stamp}`` where the stamp is
 * ``YYYYMMDDTHHMMSSZ`` (ISO-8601 compact, second precision).
 */
export function suggestMongoBackupTaskName(
  serviceName?: string | null,
  now: Date = new Date()
): string {
  const stamp = now
    .toISOString()
    .replace(/[-:]/g, '')
    .replace(/\.\d{3}Z$/, 'Z');
  const base = slugifyServiceName(serviceName) || 'mongodb';
  return `${base}-backup-${stamp}`;
}

/**
 * Return whether ``current`` is still an auto-generated / schema default value
 * that the suggestion helper may overwrite.
 */
export function isAutoMongoBackupTaskName(
  current: string,
  previousAuto: string | undefined,
  schemaDefault?: string
): boolean {
  const trimmed = current.trim();
  const schemaTrimmed = schemaDefault?.trim();
  return (
    trimmed === '' ||
    (schemaTrimmed !== undefined &&
      schemaTrimmed !== '' &&
      trimmed === schemaTrimmed) ||
    (previousAuto !== undefined && trimmed === previousAuto)
  );
}
