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

import type { SxProps, Theme } from '@mui/material/styles';

/** Matches the backend ``DetailHighlightLanguage`` enum. */
export type DetailSyntaxLanguage = 'sql' | 'json' | 'yaml' | 'bash';

/** Shared layout for syntax blocks and matching Suspense fallbacks (no prism/sql deps). */
export const detailSyntaxBlockSx: SxProps<Theme> = {
  fontFamily: "'Roboto Mono', ui-monospace, monospace",
  fontSize: '0.8125rem',
  lineHeight: 1.6,
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-word',
  p: 2,
  mt: 0.5,
  borderRadius: 1,
  bgcolor: 'action.hover',
  border: 1,
  borderColor: 'divider',
  overflowX: 'auto',
};
