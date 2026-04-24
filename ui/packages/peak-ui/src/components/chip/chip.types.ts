import type { ChipProps as MuiChipProps } from '@mui/material/Chip';

declare module '@mui/material/Chip' {
  interface ChipPropsColorOverrides {
    primary: false;
    secondary: false;
  }
}

export type ChipProps = Omit<MuiChipProps, 'icon'>;
