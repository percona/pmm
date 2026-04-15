import { Box, CircularProgress, Stack } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { useBootstrap } from 'hooks/utils/useBootstrap';
import { Sidebar } from 'components/sidebar';
import { GrafanaPage } from 'pages/grafana';
import { useGrafana } from 'contexts/grafana';
import { UpdateModal } from 'components/main/update-modal';
import { DelayedRender } from 'components/delayed-render';
import { AdreChatWidget } from 'components/adre/AdreChatWidget';
import { SHOW_UPDATE_INFO_DELAY_MS } from 'lib/constants';
import { isRenderingServer } from '@pmm/shared';
import Header from './header/Header';

export const MainWithNav = () => {
  const { isReady } = useBootstrap();
  const { isFullScreen, isOnGrafanaPage } = useGrafana();

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
    <Stack direction="row" flex={1} sx={{ minHeight: 0, minWidth: 0, height: '100%', width: '100%' }}>
      {!isFullScreen && !isRenderingServer() && <Sidebar />}
      <Stack flex={1} direction="column" sx={{ minHeight: 0, minWidth: 0, height: '100%', width: '100%' }}>
        {!isFullScreen && <Header />}
        <Box
          sx={{
            flex: '1 1 0%',
            minHeight: 0,
            minWidth: 0,
            width: '100%',
            display: isOnGrafanaPage ? 'none' : 'flex',
            flexDirection: 'column',
            overflow: 'auto',
          }}
        >
          <Outlet />
        </Box>
        <GrafanaPage />
      </Stack>
      <DelayedRender delay={SHOW_UPDATE_INFO_DELAY_MS}>
        <UpdateModal />
      </DelayedRender>
      {!isFullScreen && <AdreChatWidget />}
    </Stack>
  );
};
