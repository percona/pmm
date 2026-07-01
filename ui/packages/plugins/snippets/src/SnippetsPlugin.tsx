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

import { Route, Routes } from 'react-router-dom';
import { SnippetDetailPage } from './SnippetDetailPage';
import { SnippetsListPage } from './SnippetsListPage';

interface SnippetsPluginProps {
  /** Whether the current user has admin privileges. Controls approval UI visibility. */
  isAdmin?: boolean;
}

/**
 * Snippets plugin entry point — composes its own routes because the legacy
 * snippets UI is snippet-centric (list of files → detail page combining
 * preview + execution form + history) rather than the framework's
 * task-centric default.
 *
 * Pass `isAdmin` from the shell's auth context to enable per-row and
 * batch approval controls for admin users.
 */
export function SnippetsPlugin({ isAdmin = false }: SnippetsPluginProps) {
  return (
    <Routes>
      <Route index element={<SnippetsListPage isAdmin={isAdmin} />} />
      <Route path=":filename" element={<SnippetDetailPage />} />
    </Routes>
  );
}
