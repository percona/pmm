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

import { cloneElement, useId, type ReactElement, type ReactNode } from 'react';
import { Box, Tooltip } from '@mui/material';
import { useAuth } from '@pmm-extensions/api';

/** Why an action is unavailable to a reader who may look but not act. */
export const ADMIN_REQUIRED = 'Admin role required';

/**
 * Disables an admin-only action for anyone else. The focusable wrapper carries the reason,
 * since a disabled button cannot take focus.
 */
export const AdminAction = ({
  title,
  children,
}: {
  title?: ReactNode;
  children: ReactElement<{ disabled?: boolean; 'aria-describedby'?: string }>;
}) => {
  const { canMutate } = useAuth();
  const reasonId = useId();

  // The span is load-bearing: MUI disables pointer events on a disabled button, so a
  // Tooltip on the button itself never opens while it is disabled.
  if (canMutate) {
    return title ? (
      <Tooltip title={title}>
        <Box component="span">{children}</Box>
      </Tooltip>
    ) : (
      children
    );
  }
  return (
    <Tooltip title={ADMIN_REQUIRED}>
      <Box component="span" tabIndex={0}>
        {cloneElement(children, {
          disabled: true,
          'aria-describedby': reasonId,
        })}
        <span id={reasonId} hidden>
          {ADMIN_REQUIRED}
        </span>
      </Box>
    </Tooltip>
  );
};
