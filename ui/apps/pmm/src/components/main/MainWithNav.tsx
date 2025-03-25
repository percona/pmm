import { CircularProgress, Stack } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { useBootstrap } from 'hooks/utils/useBootstrap';
import { Sidebar } from 'components/sidebar';
import { GrafanaPage } from 'pages/grafana';

export const MainWithNav = () => {
  const { isReady } = useBootstrap();

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
    <Stack direction="row" flex={1}>
      <Sidebar />
      <Outlet />
      <GrafanaPage />
    </Stack>
  );
};
