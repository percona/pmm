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
import { Alert, AlertTitle, Box } from '@mui/material';

/** Where a failure is shown, which decides how it is framed. */
export type OmErrorPlacement = 'load' | 'action' | 'item';

// One line high beside a control or in a row: the title runs into the message
// instead of sitting on a line of its own.
const COMPACT_SX = {
  py: 0,
  px: 1,
  alignItems: 'center',
  width: 'fit-content',
  '& .MuiAlert-icon': { fontSize: 18, py: 0.5, mr: 1 },
  '& .MuiAlert-message': { py: 0.5 },
};

/**
 * The one way Operations shows that something went wrong.
 *
 * The rule, for every surface in this plugin:
 *
 * - **load** - a page or section could not be read. Shown in place of what could not
 *   be loaded, full width, and says what that was.
 * - **action** - something the reader asked for did not happen, or cannot until
 *   something is fixed. Shown beside the control that asked, compact, and kept until
 *   the next attempt or until it no longer holds.
 * - **item** - one row's or one dialog's own problem. Shown in that row or dialog,
 *   compact, never lifted to the page.
 *
 * **Every failure, never just the first**: each message is shown, as a list when there
 * is more than one, so fixing one never uncovers another the reader was not told about.
 * **Never bare coloured text**: an error is always in this frame, so it reads as an
 * error in any colour scheme and to a screen reader, which announces it as an alert.
 */
export const OmError = ({
  placement,
  title,
  messages,
  severity = 'error',
  action,
  itemTestId,
}: {
  placement: OmErrorPlacement;
  /** What failed, e.g. "Could not start a scan". */
  title?: ReactNode;
  /** Why, one entry per failure. Empty entries are dropped. */
  messages?: ReactNode | ReactNode[];
  /** `warning` for a partial failure, `info` for an expected refusal. */
  severity?: 'error' | 'warning' | 'info';
  /** A control to act on it with, such as a link to what it is about. */
  action?: ReactNode;
  /** Test id put on each message, for a test counting them. */
  itemTestId?: string;
}) => {
  const list = (Array.isArray(messages) ? messages : [messages]).filter(
    (message) => message !== null && message !== undefined && message !== ''
  );
  const compact = placement !== 'load';
  return (
    <Alert
      severity={severity}
      variant={compact ? 'outlined' : 'standard'}
      data-testid="om-error"
      data-placement={placement}
      action={action}
      sx={compact ? COMPACT_SX : undefined}
    >
      {title &&
        (compact ? (
          <Box component="span" sx={{ fontWeight: 'fontWeightMedium' }}>
            {title}
            {list.length === 1 && ': '}
          </Box>
        ) : (
          <AlertTitle>{title}</AlertTitle>
        ))}
      {list.length === 1 ? (
        <Box component="span" data-testid={itemTestId}>
          {list[0]}
        </Box>
      ) : (
        list.length > 1 && (
          <Box component="ul" sx={{ m: 0, pl: 2.5 }}>
            {list.map((message, index) => (
              <li key={index} data-testid={itemTestId}>
                {message}
              </li>
            ))}
          </Box>
        )
      )}
    </Alert>
  );
};
