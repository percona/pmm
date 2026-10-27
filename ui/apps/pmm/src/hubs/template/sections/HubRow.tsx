import Box from '@mui/material/Box';
import type { FC, ReactNode } from 'react';
import { HUB_CONTAINER } from './hub.sx';

/** From this hub width, a row splits into its two columns. */
export const HUB_TWO_COLUMNS_FROM = 1024;

export interface HubRowProps {
  /** Without it, the right side takes both columns. */
  left?: ReactNode;
  /** Sections stack here when the row has more than one on this side. */
  right?: ReactNode;
  /** Both sides take the taller one's height, so boxes in them can match. */
  stretch?: boolean;
}

const cell = { display: 'flex', flexDirection: 'column', gap: 4, minWidth: 0 };

/** One row of the hub under the status banner: two equal columns with a 24px gap, one column below 1024px of the hub's width. */
export const HubRow: FC<HubRowProps> = ({ left, right, stretch = false }) => (
  <Box
    sx={(theme) => ({
      display: 'grid',
      gridTemplateColumns: 'minmax(0, 1fr)',
      alignItems: stretch ? 'stretch' : 'start',
      // Stacked, the cells keep the rows' 32px rhythm.
      rowGap: 4,
      columnGap: 3,
      [theme.containerQueries(HUB_CONTAINER).up(HUB_TWO_COLUMNS_FROM)]: {
        gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
      },
    })}
  >
    {left ? (
      <>
        <Box sx={cell}>{left}</Box>
        {/* An empty side holds its column without adding a gap when stacked. */}
        <Box sx={{ ...cell, '&:empty': { display: 'none' } }}>{right}</Box>
      </>
    ) : (
      <Box sx={{ ...cell, gridColumn: '1 / -1' }}>{right}</Box>
    )}
  </Box>
);
