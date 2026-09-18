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

import LinkIcon from '@mui/icons-material/Link';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';

export interface ChainDisplayProps {
  chainNames?: readonly string[] | null;
  /** When chainNames is empty but the task is part of a chain, depth > 0 renders a fallback link icon. */
  chainDepth?: number | null;
  onChainItemClick?: (name: string, index: number) => void;
}

export function ChainDisplay({
  chainNames,
  chainDepth,
  onChainItemClick,
}: ChainDisplayProps) {
  if (chainNames && chainNames.length > 0) {
    return (
      <Box
        component="span"
        sx={{
          display: 'inline-flex',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: 0.5,
        }}
        data-testid="chain-display"
      >
        {chainNames.map((name, index) => (
          <Box
            key={`${name}-${index}`}
            component="span"
            sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
          >
            <Chip
              size="small"
              label={name}
              variant="outlined"
              clickable={!!onChainItemClick}
              onClick={
                onChainItemClick
                  ? () => onChainItemClick(name, index)
                  : undefined
              }
            />
            {index < chainNames.length - 1 && (
              <Typography component="span" variant="body2" aria-hidden>
                →
              </Typography>
            )}
          </Box>
        ))}
      </Box>
    );
  }

  if (typeof chainDepth === 'number' && chainDepth > 0) {
    return (
      <Tooltip title={`Chained task (depth: ${chainDepth})`}>
        <LinkIcon fontSize="small" data-testid="chain-depth-icon" />
      </Tooltip>
    );
  }

  return (
    <Typography component="span" variant="body2" color="text.secondary">
      —
    </Typography>
  );
}
