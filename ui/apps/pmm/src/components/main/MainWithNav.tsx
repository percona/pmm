import { CircularProgress, Stack } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { useBootstrap } from 'hooks/utils/useBootstrap';
import { Sidebar } from 'components/sidebar';
import { GrafanaPage } from 'pages/grafana';
import { useGrafana } from 'contexts/grafana';
import { UpdateModal } from 'components/main/update-modal';
import { DelayedRender } from 'components/delayed-render';
import { SHOW_UPDATE_INFO_DELAY_MS } from 'lib/constants';
import Header from './header/Header';

export const MainWithNav = () => {
  const { isReady } = useBootstrap();
  const { isFullScreen } = useGrafana();

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
      {!isFullScreen && <Sidebar />}
      <Stack flex={1} direction="column">
        {!isFullScreen && <Header />}
        <Outlet />
        <GrafanaPage />
      </Stack>
      <DelayedRender delay={SHOW_UPDATE_INFO_DELAY_MS}>
        <UpdateModal />
      </DelayedRender>
    </Stack>
  );
};
