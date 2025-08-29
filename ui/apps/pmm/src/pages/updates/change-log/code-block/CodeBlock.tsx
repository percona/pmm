import { Typography } from '@mui/material';
import { PEAK_DARK_THEME, PEAK_LIGHT_THEME } from '@pmm/shared';
import { FC, PropsWithChildren } from 'react';

export const CodeBlock: FC<PropsWithChildren> = ({ children }) => {
  const isSingleLine =
    typeof children === 'string' && children.split('\n').length < 2;

  return (
    <Typography
      sx={[
        {
          backgroundColor: PEAK_LIGHT_THEME.action.hover,
          fontFamily: 'Roboto Mono, monospace',
          whiteSpace: 'pre',
        },
        (theme) =>
          isSingleLine
            ? {
                display: 'inline-block',
              }
            : {
                py: 1,
                px: 1.5,
                border: 2,
                borderColor: theme.palette.divider,
                borderRadius: theme.shape.borderRadius / 4,
              },
        (theme) =>
          theme.applyStyles('dark', {
            backgroundColor: PEAK_DARK_THEME.action.hover,
          }),
      ]}
    >
      {children}
    </Typography>
  );
};
