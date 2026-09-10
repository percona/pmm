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
 * Snippets plugin JSON API path prefix (axios `apiClient` uses base `/api`).
 *
 * Per-snippet routes live under `${SNIPPETS_PLUGINS_API_BASE}/snippet/...` and
 * carry the snippet filename in the `snippet_filename` query parameter.
 * Query strings are opaque to the routing layer, so the contract
 * survives the trip end-to-end regardless of intermediate normalisation.
 */
export const SNIPPETS_PLUGINS_API_BASE = '/apps/snippets';

/** API path prefix for per-snippet operations (filename rides in the query string). */
export const SNIPPET_PLUGIN_PER_SNIPPET_BASE = `${SNIPPETS_PLUGINS_API_BASE}/snippet`;

function buildPerSnippetPath(action: string, filename: string): string {
  const params = new URLSearchParams({ snippet_filename: filename });
  return `${SNIPPET_PLUGIN_PER_SNIPPET_BASE}/${action}?${params.toString()}`;
}

/** Path for `GET` per-snippet plugin schema. */
export function snippetPluginSchemaPath(filename: string): string {
  return buildPerSnippetPath('schema', filename);
}

/** Path for `POST` snippet execution. */
export function snippetPluginExecutePath(filename: string): string {
  return buildPerSnippetPath('execute', filename);
}

/** Path for `GET` script preview. */
export function snippetPluginPreviewPath(filename: string): string {
  return buildPerSnippetPath('preview', filename);
}

/** Path for `GET` raw snippet file download. */
export function snippetPluginDownloadPath(filename: string): string {
  return buildPerSnippetPath('download', filename);
}

/** Path for `GET` per-snippet execution history. */
export function snippetPluginHistoryPath(filename: string): string {
  return buildPerSnippetPath('history', filename);
}

/** Path for `PUT`/`DELETE` snippet approval. */
export function snippetPluginApprovalPath(filename: string): string {
  return buildPerSnippetPath('approval', filename);
}
