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

import { useState, type ReactNode } from 'react';
import { IconButton, Menu, Tooltip } from '@mui/material';
import MoreVertIcon from '@mui/icons-material/MoreVert';

/**
 * A row's secondary actions, behind one ellipsis.
 *
 * Three text buttons in a row's action column do not fit the width a page gets at
 * 1440 with the nav open, and the one that fell off the right edge was the red
 * Forget - which is the row-action half of P17 exactly. Widening the column cannot
 * fix it: the buttons are text, and there are three.
 *
 * So the routine actions stay inline and the rest move here, which is also what P14
 * asks for Forget on its own account: an overflow menu rather than a red button
 * sitting permanently beside the things an operator uses daily.
 */
export const RowOverflowMenu = ({
  label,
  children,
}: {
  /** What the menu is for, for a screen reader. */
  label: string;
  /** `MenuItem`s. They receive no close handler, so each should call `onClose`. */
  children: (close: () => void) => ReactNode;
}) => {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const close = () => setAnchor(null);
  return (
    <>
      <Tooltip title={label}>
        <IconButton
          size="small"
          aria-label={label}
          onClick={(event) => setAnchor(event.currentTarget)}
        >
          <MoreVertIcon fontSize="small" />
        </IconButton>
      </Tooltip>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={close}>
        {children(close)}
      </Menu>
    </>
  );
};
