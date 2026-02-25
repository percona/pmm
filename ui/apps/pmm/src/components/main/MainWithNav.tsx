import { CircularProgress, Stack } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { useBootstrap } from 'hooks/utils/useBootstrap';
import { Sidebar } from 'components/sidebar';
import { GrafanaPage } from 'pages/grafana';
import { useGrafana } from 'contexts/grafana';
import { UpdateModal } from 'components/main/update-modal';
import { DelayedRender } from 'components/delayed-render';
import { SHOW_UPDATE_INFO_DELAY_MS } from 'lib/constants';
import { useMemo } from 'react';

export const MainWithNav = () => {
  const { isReady } = useBootstrap();
  const { isOnGrafanaPage, isFullScreen } = useGrafana();
  // We hide the sidebar in headless browser to avoid the navigation to be shown on rendering server
  // Checking the NODE_ENV to avoid the sidebar vanishing when running tests (e.g. Playwright)
  const isHeadlessBrowser = useMemo(() => {
    return ['development', 'production'].includes(process.env.NODE_ENV || '') && window.navigator.userAgent.includes('Headless');
  }, []);

  if (!isReady) {
    return (
      <Stack
        alignItems="center"
        justifyContent="center"
        sx={{
          flex: 1,
          padding: 10,
        }}
      >
        <CircularProgress data-testid="pmm-loading-indicator" />
      </Stack>
    );
  }

  return (
    <Stack direction="row" flex={1}>
      {!isFullScreen && (
        <>
          {!isHeadlessBrowser && <Sidebar />}
          <Stack flex={isOnGrafanaPage ? 0 : 1} direction="column">
            <Outlet />
          </Stack>
        </>
      )}
      <GrafanaPage />
      <DelayedRender delay={SHOW_UPDATE_INFO_DELAY_MS}>
        <UpdateModal />
      </DelayedRender>
    </Stack>
  );
};
