import Box from '@mui/material/Box';
import type { FC, ReactNode } from 'react';

export interface HubBandProps {
  /** Widest the band's content gets, in px; the hub's full width when absent. */
  maxWidth?: number;
  children: ReactNode;
}

/** One row of the hub under its header, as the Figma hub's sections: 16px above and below, centered under its own width. */
export const HubBand: FC<HubBandProps> = ({ maxWidth, children }) => (
  <Box sx={{ width: '100%', maxWidth, mx: 'auto', py: 2 }}>{children}</Box>
);
