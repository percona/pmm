import { Box, CircularProgress } from '@mui/material';
import { FC } from 'react';
import { STATUS_STYLES } from '../investigationStyles';

/** An investigation's status as a colored pill; a spinner replaces the dot while it runs. */
export const StatusPill: FC<{ status: string; running?: boolean }> = ({
  status,
  running,
}) => {
  const style = STATUS_STYLES[status] ?? STATUS_STYLES.open;
  return (
    <Box
      component="span"
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 0.75,
        px: 1.25,
        py: 0.25,
        borderRadius: 999,
        bgcolor: style.bg,
        color: style.color,
        fontSize: 12,
        fontWeight: 500,
        whiteSpace: 'nowrap',
      }}
    >
      {running ? (
        <CircularProgress size={10} sx={{ color: 'inherit' }} />
      ) : (
        <Box
          component="span"
          sx={{
            width: 7,
            height: 7,
            borderRadius: '50%',
            bgcolor: 'currentColor',
          }}
        />
      )}
      {status.replace('_', ' ')}
    </Box>
  );
};
