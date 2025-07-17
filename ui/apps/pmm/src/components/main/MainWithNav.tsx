import { CircularProgress, Stack } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { useBootstrap } from 'hooks/utils/useBootstrap';
import { Sidebar } from 'components/sidebar';
import { GrafanaPage } from 'pages/grafana';
import { useKioskMode } from 'hooks/utils/useKioskMode';
import { useGrafana } from 'contexts/grafana';

export const MainWithNav = () => {
  const { isReady } = useBootstrap();
  const kioskMode = useKioskMode();
  const { isOnGrafanaPage } = useGrafana();

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
      {!kioskMode.active && (
        <>
          <Sidebar />
          <Stack flex={isOnGrafanaPage ? 0 : 1} direction="column">
            <Outlet />
          </Stack>
        </>
      )}
      <GrafanaPage />
    </Stack>
  );
};
