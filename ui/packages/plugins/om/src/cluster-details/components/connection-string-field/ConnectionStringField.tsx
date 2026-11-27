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

import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { CopyToClipboardButton } from '@percona/peak-ui';
import { Messages } from '../../ClusterDetails.messages';

export const ConnectionStringField = ({ value }: { value: string }) => (
  <Stack
    direction="row"
    alignItems="center"
    sx={{
      border: 1,
      borderColor: 'divider',
      borderRadius: 1,
      maxWidth: 560,
      minWidth: 0,
    }}
    data-testid="om-connection-string"
  >
    <Typography
      variant="body2"
      color="text.secondary"
      sx={{
        px: 1.5,
        py: 0.75,
        borderRight: 1,
        borderColor: 'divider',
        whiteSpace: 'nowrap',
      }}
    >
      {Messages.connectionString.label}
    </Typography>
    <Tooltip title={value}>
      <Box
        component="code"
        tabIndex={0}
        sx={{
          px: 1.5,
          flex: 1,
          minWidth: 0,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          fontFamily: 'monospace',
          fontSize: 'body2.fontSize',
        }}
        data-testid="om-connection-string-value"
      >
        {value}
      </Box>
    </Tooltip>
    <Box sx={{ borderLeft: 1, borderColor: 'divider', px: 0.5 }}>
      <CopyToClipboardButton
        textToCopy={value}
        showCopyButtonText
        copyCommand={Messages.connectionString.copy}
      />
    </Box>
  </Stack>
);
