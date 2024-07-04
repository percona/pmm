import { CircularProgress, Stack } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { AppBar } from '../app-bar/AppBar';
import { useAuth } from 'contexts/auth';

export const Main = () => {
  const { isLoading } = useAuth();

  if (isLoading) {
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
