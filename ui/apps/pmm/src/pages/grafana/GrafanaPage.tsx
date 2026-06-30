import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import { useGrafana } from 'contexts/grafana';
import { PMM_BASE_PATH, PMM_HOME_URL } from 'lib/constants';
import messenger from 'lib/messenger';
import { constructUrl } from 'utils/link.utils';
import { FC, useCallback, useEffect, useMemo, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { isGrafanaLoginPath } from 'contexts/auth/auth.clientSession';
import { handleGrafanaUserLoggedOut } from 'contexts/auth/auth.grafanaLogout';
import { GrafanaPageFrame } from 'components/grafana-page-frame';
import {
  getIframePathname,
  redirectIframeFromPmmShell,
} from './grafanaIframe.utils';

export const GrafanaPage: FC = () => {
  const queryClient = useQueryClient();
  const { isFrameLoaded, isOnGrafanaPage, frameRef, isFullScreen } =
    useGrafana();
  const src = useMemo(
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

  useEffect(() => {
    if (!isFrameLoaded) {
      return;
    }

    messenger.waitForMessage('GRAFANA_READY', 5_000).finally(() => {
      setLoading(false);
    });
  }, [isFrameLoaded]);

  const handleIframeLoad = useCallback(() => {
    const iframe = frameRef?.current;
    if (isGrafanaLoginPath(getIframePathname(iframe))) {
      handleGrafanaUserLoggedOut(queryClient);
      return;
    }
    if (iframe) {
      redirectIframeFromPmmShell(iframe, src);
    }
  }, [frameRef, queryClient, src]);

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
        <GrafanaPageFrame>
          <Box
            key={src}
            id="grafana-iframe"
            ref={frameRef}
            src={src}
            component="iframe"
            onLoad={handleIframeLoad}
            sx={
              isFullScreen
                ? { border: 'none', flex: 1 }
                : { flex: 1, border: 0 }
            }
          />
        </GrafanaPageFrame>
      </Stack>
    </>
  );
};
