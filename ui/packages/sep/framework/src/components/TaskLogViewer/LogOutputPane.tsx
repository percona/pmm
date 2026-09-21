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

import { LazyLog } from '@melloware/react-logviewer';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';

/**
 * Keep LazyLog's search bar from announcing a result count nobody asked for.
 *
 * The bar always renders `0 matches`, before a single character is typed — the
 * library offers no prop to suppress it and no hook to tell "not searched yet"
 * from "searched, found nothing". The input's own emptiness is the signal, and
 * `:placeholder-shown` is the only way to read it from CSS.
 *
 * `visibility` rather than `display`, so the bar's controls do not shift
 * sideways the moment a search begins. A genuine zero-result search still
 * reports itself: once the box holds text, the count reappears.
 *
 * A single typed character is still "before a search" to the library, whose
 * `searchMinCharacters` is 2 — that case keeps showing `0 matches`. CSS cannot
 * count characters, and the state lasts one keystroke.
 */
const HIDE_UNSEARCHED_MATCH_COUNT = {
  '& .react-lazylog-searchbar:has(.react-lazylog-searchbar-input:placeholder-shown) .react-lazylog-searchbar-matches':
    { visibility: 'hidden' },
} as const;

export interface LogOutputPaneProps {
  text: string;
  wrap: boolean;
  enableSearch?: boolean;
  height?: number | string;
  emptyLabel?: string;
  /**
   * Whether the pane sticks to the last line as text arrives. Off by default:
   * a finished report is read from the top, and only a run still producing
   * output wants tailing.
   */
  follow?: boolean;
}

export function LogOutputPane({
  text,
  wrap,
  enableSearch = true,
  height = 400,
  emptyLabel = 'No output yet.',
  follow = false,
}: LogOutputPaneProps) {
  if (!text) {
    return (
      <Box sx={{ height, width: '100%', p: 2, color: 'text.secondary' }}>
        <Typography variant="body2">{emptyLabel}</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ height, width: '100%', ...HIDE_UNSEARCHED_MATCH_COUNT }}>
      <LazyLog
        text={text}
        extraLines={1}
        enableSearch={enableSearch}
        wrapLines={wrap}
        follow={follow}
        selectableLines
        caseInsensitive
      />
    </Box>
  );
}
