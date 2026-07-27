import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import { useGrafana } from 'contexts/grafana';
import { useMessengerListener } from 'hooks/utils/useMessengerListener';
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

  useMessengerListener('GRAFANA_READY', () => {
    setLoading(false);
    // Answer the handshake so Grafana's own outbox drains too.
    messenger.sendMessage({ type: 'MESSENGER_READY' });
  });

  // Show the frame anyway if Grafana never announces itself.
  useEffect(() => {
    if (!isFrameLoaded) {
      return;
    }

    setLoading(true);
    const timeoutId = setTimeout(() => setLoading(false), 5_000);

    return () => clearTimeout(timeoutId);
  }, [isFrameLoaded, src]);

  const handleIframeLoad = useCallback(() => {
    const iframe = frameRef?.current;
    // The frame may have navigated in place, which keeps its window identity —
    // make the messenger wait for a fresh handshake before trusting the peer.
    messenger.invalidateTarget();
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
