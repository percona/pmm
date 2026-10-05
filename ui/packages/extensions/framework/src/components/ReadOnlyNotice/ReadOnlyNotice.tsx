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

import LockIcon from '@mui/icons-material/Lock';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';

export interface ReadOnlyNoticeProps {
  /** What the session cannot change, e.g. `"create tasks"`. Defaults to a generic phrasing. */
  action?: string;
  /**
   * `page` (default) fills a whole route with the guard state. `inline`
   * renders the sentence alone, for a withheld control inside a larger page
   * that stays useful without it.
   */
  variant?: 'page' | 'inline';
  testId?: string;
}

/**
 * State for a mutation surface a read-only session reached anyway — by direct
 * URL entry, a bookmark, or a role that changed mid-session. The entry points
 * to these surfaces are hidden, so this is the backstop that keeps a blank
 * screen from being the answer.
 *
 * Single home for the wording, so the page guard and an inline withheld control
 * cannot drift apart. It is not a security boundary: the backend gate is.
 */
export function ReadOnlyNotice({
  action = 'make changes here',
  variant = 'page',
  testId = 'read-only-notice',
}: ReadOnlyNoticeProps) {
  const sentence = (
    <Typography variant="body2" color="text.secondary">
      You don&apos;t have permission to {action}.
    </Typography>
  );

  if (variant === 'inline') {
    return <Box data-testid={testId}>{sentence}</Box>;
  }

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        py: 10,
        textAlign: 'center',
      }}
      data-testid={testId}
    >
      <LockIcon sx={{ fontSize: 64, color: 'text.secondary', mb: 2 }} />
      <Typography variant="h5" gutterBottom>
        Read-only access
      </Typography>
      {sentence}
    </Box>
  );
}
