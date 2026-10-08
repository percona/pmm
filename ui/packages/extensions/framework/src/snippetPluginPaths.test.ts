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
import {
  SNIPPET_PLUGIN_PER_SNIPPET_BASE,
  snippetPluginApprovalPath,
  snippetPluginDownloadPath,
  snippetPluginExecutePath,
  snippetPluginHistoryPath,
  snippetPluginPreviewPath,
  snippetPluginSchemaPath,
} from './snippetPluginPaths';

describe('snippetPluginPaths', () => {
  it('exposes the /snippet sub-prefix for per-snippet operations', () => {
    expect(SNIPPET_PLUGIN_PER_SNIPPET_BASE).toBe('/apps/snippets/snippet');
  });

  it('carries the filename in the snippet_filename query parameter', () => {
    expect(snippetPluginSchemaPath('hello.sh')).toBe(
      '/apps/snippets/snippet/schema?snippet_filename=hello.sh'
    );
    expect(snippetPluginExecutePath('hello.sh')).toBe(
      '/apps/snippets/snippet/execute?snippet_filename=hello.sh'
    );
    expect(snippetPluginPreviewPath('hello.sh')).toBe(
      '/apps/snippets/snippet/preview?snippet_filename=hello.sh'
    );
    expect(snippetPluginDownloadPath('hello.sh')).toBe(
      '/apps/snippets/snippet/download?snippet_filename=hello.sh'
    );
    expect(snippetPluginHistoryPath('hello.sh')).toBe(
      '/apps/snippets/snippet/history?snippet_filename=hello.sh'
    );
    expect(snippetPluginApprovalPath('hello.sh')).toBe(
      '/apps/snippets/snippet/approval?snippet_filename=hello.sh'
    );
  });

  it('escapes nested filenames using URLSearchParams encoding', () => {
    const filename = 'diag/slow-query.sh';
    expect(snippetPluginSchemaPath(filename)).toBe(
      '/apps/snippets/snippet/schema?snippet_filename=diag%2Fslow-query.sh'
    );
    expect(snippetPluginExecutePath(filename)).toBe(
      '/apps/snippets/snippet/execute?snippet_filename=diag%2Fslow-query.sh'
    );
  });

  it('never bakes nested filenames into the URL path itself', () => {
    const filename = 'diag/slow-query.sh';
    for (const builder of [
      snippetPluginSchemaPath,
      snippetPluginExecutePath,
      snippetPluginPreviewPath,
      snippetPluginDownloadPath,
      snippetPluginHistoryPath,
      snippetPluginApprovalPath,
    ]) {
      const url = builder(filename);
      const [path] = url.split('?');
      expect(path).not.toContain('diag');
      expect(path).not.toContain('slow-query');
      expect(path).not.toContain('%2F');
    }
  });
});
