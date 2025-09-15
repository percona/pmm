import { CircularProgress, Stack } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { useBootstrap } from 'hooks/utils/useBootstrap';
import { Sidebar } from 'components/sidebar';
import { GrafanaPage } from 'pages/grafana';
import { useGrafana } from 'contexts/grafana';
import { UpdateModal } from 'components/update-modal';

export const MainWithNav = () => {
  const { isReady } = useBootstrap();
  const { isOnGrafanaPage, isFullScreen } = useGrafana();

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
          <Sidebar />
          <Stack flex={isOnGrafanaPage ? 0 : 1} direction="column">
            <Outlet />
          </Stack>
        </>
      )}
      <GrafanaPage />
      <UpdateModal />
    </Stack>
  );
};
