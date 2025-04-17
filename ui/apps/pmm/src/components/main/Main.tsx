import { CircularProgress, Stack } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { AppBar } from '../app-bar/AppBar';
import { useBootstrap } from 'hooks/utils/useBootstrap';

export const Main = () => {
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
    <Stack>
      <AppBar />
      <Outlet />
    </Stack>
  );
};
