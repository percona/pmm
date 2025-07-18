import { Typography } from '@mui/material';
import { FC, PropsWithChildren } from 'react';

export const CodeBlock: FC<PropsWithChildren> = ({ children }) => {
  const isSingleLine =
    typeof children === 'string' && children.split('\n').length < 2;

  return (
    <Typography
      sx={[
        (theme) => ({
          p: isSingleLine ? undefined : 1,
          display: isSingleLine ? 'inline-block' : undefined,
          border: isSingleLine ? undefined : 1,
          borderColor: isSingleLine ? undefined : theme.palette.divider,
          borderRadius: isSingleLine ? 'none' : theme.shape.borderRadius / 4,
          fontFamily: 'Roboto Mono, monospace',
          whiteSpace: 'pre',
          backgroundColor: theme.palette.grey[200],
        }),
        (theme) =>
          theme.applyStyles('dark', {
            backgroundColor: theme.palette.surfaces?.high,
          }),
      ]}
    >
      {children}
    </Typography>
  );
};
