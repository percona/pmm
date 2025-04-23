import { Box, Stack } from '@mui/material';
import { useGrafana } from 'contexts/grafana';
import { PMM_BASE_PATH, PMM_HOME_URL } from 'lib/constants';
import { FC, useMemo } from 'react';

export const GrafanaPage: FC = () => {
  const { isFrameLoaded, isOnGrafanaPage, frameRef } = useGrafana();
  const src = useMemo(
    // load specific grafana page as the first one
    () =>
      isFrameLoaded
        ? window.location.pathname.replace(PMM_BASE_PATH, '')
        : PMM_HOME_URL,
    [isFrameLoaded]
  );

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
        id="grafana-iframe"
        ref={frameRef}
        src={src}
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
