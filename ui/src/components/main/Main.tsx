import { CircularProgress, Stack } from '@mui/material';
import { Outlet, useLocation } from 'react-router-dom';
import { AppBar } from '../app-bar/AppBar';
import { useBootstrap } from 'hooks/utils/useBootstrap';
import { Grafana } from 'components/grafana/Grafana';
import { SideNav } from 'components/sidenav/SideNav';
import { MessagesProvider } from 'contexts/messages/messages.provider';

export const Main = () => {
  const { isReady } = useBootstrap();
  const location = useLocation();
  const isGrafana = isGrafanaPage(location.pathname);

  if (!isReady) {
    return (
      <Stack
        alignItems="center"
        justifyContent="center"
        sx={{
          padding: 10,
        }}
      >
        <CircularProgress data-testid="pmm-loading-indicator" />
      </Stack>
    );
  }

  return (
    <MessagesProvider>
      <Stack>
        <AppBar />
        <Stack direction="row">
          <SideNav />
          {!isGrafana && <Outlet />}
          <Stack
            sx={{
              flex: 1,
              visibility: isGrafana ? 'visible' : 'hidden',
              width: isGrafana ? 'auto' : 0,
            }}
          >
            <Grafana
              url={
                isGrafana
                  ? `/graph${location.pathname}?${location.search}`
                  : '/graph'
              }
            />
          </Stack>
        </Stack>
      </Stack>
    </MessagesProvider>
  );
};

const isGrafanaPage = (pathname: string) => {
  return pathname.startsWith('/d') || pathname.startsWith('/alerts');
};
