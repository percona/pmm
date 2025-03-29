import { Typography } from '@mui/material';
import { FC, PropsWithChildren } from 'react';

export const CodeBlock: FC<PropsWithChildren> = ({ children }) => {
  const isSingleLine =
    typeof children === 'string' && children.split('\n').length < 2;

  return (
    <Typography
      sx={(theme) => ({
        display: isSingleLine ? 'inline-block' : undefined,
        fontFamily: 'Roboto Mono, monospace',
        whiteSpace: 'pre',
        backgroundColor: theme.palette.grey[200],
      })}
    >
      {children}
    </Typography>
  );
};
