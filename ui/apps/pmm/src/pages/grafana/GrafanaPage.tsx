import { Box, CircularProgress, Stack } from '@mui/material';
import { useGrafana } from 'contexts/grafana';
import { useKioskMode } from 'hooks/utils/useKioskMode';
import { PMM_BASE_PATH, PMM_HOME_URL } from 'lib/constants';
import messenger from 'lib/messenger';
import { constructUrl } from 'lib/utils/link.utils';
import { FC, useEffect, useMemo, useState } from 'react';

export const GrafanaPage: FC = () => {
  const { isFrameLoaded, isOnGrafanaPage, frameRef } = useGrafana();
  const src = useMemo(
    // load specific grafana page as the first one
    () =>
      isFrameLoaded
        ? constructUrl({
            ...window.location,
            pathname: window.location.pathname.replace(PMM_BASE_PATH, ''),
          })
        : PMM_HOME_URL,
    [isFrameLoaded]
  );
  const [loading, setLoading] = useState(true);
  const kioskMode = useKioskMode();

  useEffect(() => {
    if (isFrameLoaded) {
      messenger
        .waitForMessage('GRAFANA_READY', 5_000)
        .finally(() => setLoading(false));
    }
  }, [isFrameLoaded]);

  if (!isFrameLoaded) {
    return null;
  }

  return (
    <>
      {loading && (
        <Stack
          alignItems="center"
          justifyContent="center"
          sx={{
            flex: 1,
            padding: 10,
          }}
        >
          <CircularProgress data-testid="pmm-grafana-iframe-loading-indicator" />
        </Stack>
      )}
      <Stack
        sx={{
          flex: 1,
          display: isOnGrafanaPage && !loading ? 'flex' : 'none',
        }}
      >
        <Box
          id="grafana-iframe"
          ref={frameRef}
          src={src}
          component="iframe"
          sx={
            kioskMode.active
              ? {
                  border: 'none',
                  flex: 1,
                }
              : (theme) => ({
                  borderStyle: 'solid',
                  borderColor: theme.palette.divider,
                  borderRadius: theme.shape.borderRadius,
                  flex: 1,
                  m: 1,
                })
          }
        />
      </Stack>
    </>
  );
};
