import type { SxProps, Theme } from '@mui/material/styles';

// Narrow columns wrap their header instead of cutting it off.
export const WRAPPING_HEADER_SX: SxProps<Theme> = {
  '& .Mui-TableHeadCell-Content-Wrapper': { whiteSpace: 'normal' },
};
