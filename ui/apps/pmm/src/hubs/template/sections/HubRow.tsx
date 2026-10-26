import Box from '@mui/material/Box';
import type { FC, ReactNode } from 'react';
import { HUB_CONTAINER } from './hub.sx';

/** From this hub width, a row splits into its two columns. */
export const HUB_TWO_COLUMNS_FROM = 1024;

export interface HubRowProps {
  left: ReactNode;
  /** Sections stack here when the row has more than one on this side. */
  right?: ReactNode;
}

const cell = { display: 'flex', flexDirection: 'column', gap: 4, minWidth: 0 };

/** One row of the hub under the status banner: 3fr beside 2fr with a 24px gap, one column below 1024px of the hub's width. */
export const HubRow: FC<HubRowProps> = ({ left, right }) => (
  <Box
    sx={(theme) => ({
      display: 'grid',
      gridTemplateColumns: 'minmax(0, 1fr)',
      alignItems: 'start',
      // Stacked, the cells keep the rows' 32px rhythm.
      rowGap: 4,
      columnGap: 3,
      [theme.containerQueries(HUB_CONTAINER).up(HUB_TWO_COLUMNS_FROM)]: {
        gridTemplateColumns: 'minmax(0, 3fr) minmax(0, 2fr)',
      },
    })}
  >
    <Box sx={cell}>{left}</Box>
    {/* An empty side holds its column without adding a gap when stacked. */}
    <Box sx={{ ...cell, '&:empty': { display: 'none' } }}>{right}</Box>
  </Box>
);
