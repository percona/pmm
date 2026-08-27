import { CircularProgress, Stack } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { useAuth } from 'contexts/auth';
import { useBootstrap } from 'hooks/utils/useBootstrap';
import { Sidebar } from 'components/sidebar';
import { GrafanaPage } from 'pages/grafana';
import { useGrafana } from 'contexts/grafana';
import { useUser } from 'contexts/user';
import { UpdateModal } from 'components/main/update-modal';
import { DelayedRender } from 'components/delayed-render';
import { SHOW_UPDATE_INFO_DELAY_MS } from 'lib/constants';
import { isRenderingServer } from '@pmm/shared';
import Header from './header/Header';

const useMainNavVisible = () => {
  const { isLoggedIn } = useAuth();
  const { user } = useUser();
  const { isFullScreen } = useGrafana();

  return (
    (isLoggedIn || user?.isAnonymous) && !isFullScreen && !isRenderingServer()
  );
};

export const MainWithNav = () => {
  const { isReady } = useBootstrap();
  const showNav = useMainNavVisible();

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
    <Stack direction="row" flex={1} sx={{ minWidth: 0 }}>
      {showNav && <Sidebar />}
      <Stack flex={1} direction="column" sx={{ minWidth: 0 }}>
        {showNav && <Header />}
        <Outlet />
        <GrafanaPage />
      </Stack>
      <DelayedRender delay={SHOW_UPDATE_INFO_DELAY_MS}>
        <UpdateModal />
      </DelayedRender>
    </Stack>
  );
};
