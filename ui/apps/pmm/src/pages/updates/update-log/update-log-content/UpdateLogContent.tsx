import { FC } from 'react';
import { UpdateLogContentProps } from './UpdateLogContent.types';
import { Typography } from '@mui/material';

export const UpdateLogContent: FC<UpdateLogContentProps> = ({ content }) => (
  <Typography
    variant="body2"
    sx={(theme) => ({
      display: 'block',
      fontFamily: 'Roboto Mono',
      whiteSpace: 'pre',
      width: '100%',
      height: 250,
      overflowY: 'scroll',
      color: theme.palette.grey[800],
      borderWidth: 1,
      borderRadius: 2,
      borderStyle: 'solid',
      borderColor: theme.palette.divider,
      backgroundColor: theme.palette.grey[100],
      p: 1,
    })}
  >
    {content}
  </Typography>
);
