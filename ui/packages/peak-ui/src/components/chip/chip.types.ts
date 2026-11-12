import type { ChipProps as MuiChipProps } from '@mui/material/Chip';

declare module '@mui/material/Chip' {
  interface ChipPropsColorOverrides {
    primary: false;
    secondary: false;
  }
}

// Chips shouldn't carry a leading icon (for now)
export type ChipProps = Omit<MuiChipProps, 'icon'>;
