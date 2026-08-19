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

import { Box, Tooltip } from '@mui/material';
import { UNAVAILABLE_FALLBACK, UNAVAILABLE_PHRASE } from '../constants';
import type { PomUnavailableReason } from '../types';

interface UnavailableProps {
  /** Reason code from the document's `unavailable` map. */
  reason?: PomUnavailableReason | string | null;
}

/**
 * The em-dash-with-a-cause that stands in for every null POM reports.
 *
 * A null in POM means "not observable", never zero, and the document names the
 * reason per field. Rendering `0` or a blank cell would turn "we could not see
 * it" into "there is none of it" — which is the single easiest way to make this
 * page lie. Every unavailable value on screen goes through here.
 */
export function Unavailable({ reason }: UnavailableProps) {
  const phrase =
    (reason && UNAVAILABLE_PHRASE[reason as PomUnavailableReason]) ||
    UNAVAILABLE_FALLBACK;
  return (
    <Tooltip title={phrase}>
      <Box
        component="span"
        aria-label={phrase}
        sx={{ color: 'text.disabled', cursor: 'help' }}
      >
        —
      </Box>
    </Tooltip>
  );
}
