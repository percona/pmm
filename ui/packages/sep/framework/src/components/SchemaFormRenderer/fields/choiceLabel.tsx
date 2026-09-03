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

import type { ReactNode } from 'react';
import Box from '@mui/material/Box';
import Tooltip from '@mui/material/Tooltip';
import type { ChoiceOption } from '../types';

/**
 * Render a choice's label, wrapping it in a Tooltip + span when the option is
 * disabled and carries a reason. MUI tooltips never fire on a disabled element
 * directly, so the span is the hover target; `pointerEvents: 'auto'` re-enables
 * hover inside a disabled `<MenuItem>` (whose root sets `pointer-events: none`)
 * and is harmless elsewhere. Enabled options (or disabled ones with no reason)
 * render the plain label string.
 */
export function renderChoiceLabel(choice: ChoiceOption): ReactNode {
  if (choice.disabled && choice.disabled_reason) {
    return (
      <Tooltip title={choice.disabled_reason}>
        <Box component="span" sx={{ pointerEvents: 'auto' }}>
          {choice.label}
        </Box>
      </Tooltip>
    );
  }
  return choice.label;
}
