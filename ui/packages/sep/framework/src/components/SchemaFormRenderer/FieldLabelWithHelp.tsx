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
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';

export interface FieldHelpIconProps {
  /** Tooltip body shown on hover/focus; also the accessible description. */
  description: string;
  /**
   * Field label for the accessible name (`Help for ${label}`) and for
   * `data-help-for` (preferred for per-field test queries so label matchers
   * do not also hit the icon).
   */
  label: string;
}

/**
 * Info-icon tooltip for schema form field help. Non-button span so it can sit
 * inside MUI floating labels; focusable, with a per-field accessible name and
 * the description exposed via Tooltip describeChild. Omit when there is no
 * description to show.
 */
export function FieldHelpIcon({ description, label }: FieldHelpIconProps) {
  return (
    <Tooltip title={description} describeChild>
      <Box
        component="span"
        role="img"
        tabIndex={0}
        aria-label={`Help for ${label}`}
        data-help-for={label}
        sx={{
          display: 'inline-flex',
          alignItems: 'center',
          ml: 0.5,
          verticalAlign: 'middle',
          cursor: 'help',
          color: 'action.active',
        }}
        onClick={(event) => event.preventDefault()}
      >
        <HelpOutlineIcon sx={{ fontSize: 16 }} />
      </Box>
    </Tooltip>
  );
}

export interface FieldLabelWithHelpProps {
  /** Visible field label text. */
  label: string;
  /** Optional help text; when set, an info-icon tooltip is rendered next to the label. */
  description?: string;
}

/**
 * Field label with an optional info-icon tooltip from `description`.
 * Returns a plain string when undescribed; otherwise a ReactNode for MUI
 * label slots (e.g. TextInput `textFieldProps.label`).
 */
export function FieldLabelWithHelp({
  label,
  description,
}: FieldLabelWithHelpProps): ReactNode {
  if (!description) {
    return label;
  }
  return (
    <Box component="span" sx={{ display: 'inline-flex', alignItems: 'center' }}>
      {label}
      <FieldHelpIcon description={description} label={label} />
    </Box>
  );
}
