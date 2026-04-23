import type { ChipProps as MuiChipProps } from '@mui/material';

declare module '@mui/material/Chip' {
  interface ChipPropsColorOverrides {
    primary: false;
    secondary: false;
  }
}

export type ChipProps = Omit<MuiChipProps, 'icon'> & {
  dataTestId?: string;
};
