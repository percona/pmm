import { SxProps, Theme } from '@mui/material/styles';

export const mergeSx = (
  sx: (SxProps<Theme> | undefined)[]
): NonNullable<SxProps<Theme>> =>
  sx
    .filter((item): item is NonNullable<typeof item> => item !== undefined)
    .reduce((acc, curr) => ({ ...acc, ...curr }), {});
