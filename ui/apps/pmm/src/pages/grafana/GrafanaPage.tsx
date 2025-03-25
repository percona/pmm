import { Box, Stack } from '@mui/material';
import { useGrafana } from 'contexts/grafana';
import { FC } from 'react';

export const GrafanaPage: FC = () => {
  const { isFrameLoaded, isOnGrafanaPage, frameRef } = useGrafana();

  if (!isFrameLoaded) {
    return null;
  }

  return (
    <Stack
      sx={{
        flex: 1,
        display: isOnGrafanaPage ? 'flex' : 'none',
      }}
    >
      <Box
        ref={frameRef}
        src="/graph"
        component="iframe"
        sx={(theme) => ({
          borderStyle: 'solid',
          borderColor: theme.palette.divider,
          borderRadius: theme.shape.borderRadius,
          flex: 1,
          m: 1,
        })}
      />
    </Stack>
  );
};
