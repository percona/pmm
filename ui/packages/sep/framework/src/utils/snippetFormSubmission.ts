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

/** Form field names handled outside `args` for snippet execute API bodies. */
export const SNIPPET_FORM_RESERVED_FIELD_NAMES = new Set([
  'executor_host',
  'sudo',
  'script_preview',
]);

/** Body shape accepted by `POST /api/apps/snippets/{filename}/execute`. */
export interface SnippetExecutionFormPayload {
  executor_host: string;
  sudo: boolean;
  args: Record<string, unknown>;
}

/**
 * Map raw `SchemaFormRenderer` values into the snippets execute request body.
 * Strips reserved UI fields and drops empty optional args.
 */
export function buildSnippetExecutionFormPayload(
  values: Record<string, unknown>
): SnippetExecutionFormPayload {
  const args: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(values)) {
    if (SNIPPET_FORM_RESERVED_FIELD_NAMES.has(key)) {
      continue;
    }
    if (value === '' || value === undefined) {
      continue;
    }
    args[key] = value;
  }
  return {
    executor_host: String(values.executor_host ?? ''),
    sudo: Boolean(values.sudo ?? false),
    args,
  };
}
