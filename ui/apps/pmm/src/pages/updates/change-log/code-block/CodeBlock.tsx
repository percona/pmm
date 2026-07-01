import { Typography } from '@mui/material';
import { FC, PropsWithChildren } from 'react';

export const CodeBlock: FC<PropsWithChildren> = ({ children }) => {
  const isSingleLine =
    typeof children === 'string' && children.split('\n').length < 2;

  return (
    <Typography
      sx={[
        {
          fontFamily: 'Roboto Mono, monospace',
          whiteSpace: 'pre',
          color: (theme) =>
            theme.palette.mode === 'dark'
              ? theme.palette.text.primary
              : theme.palette.action.hover,
          overflowX: 'auto',
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
                borderRadius: Number(theme.shape.borderRadius) / 4,
              },
        (theme) =>
          theme.applyStyles('dark', {
            backgroundColor: theme.palette.grey[800],
          }),
      ]}
    >
      {children}
    </Typography>
  );
};
