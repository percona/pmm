import { CircularProgress, Stack } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { AppBar } from '../app-bar/AppBar';
import { useAuth } from 'contexts/auth';

export const Main = () => {
  const { isLoading } = useAuth();

  return (
    <Stack>
      <AppBar />
      {isLoading ? (
        <Stack
          alignItems="center"
          justifyContent="center"
          sx={{
            padding: 10,
          }}
        >
          <CircularProgress data-testid="pmm-loading-indicator" />
        </Stack>
      ) : (
        <Outlet />
      )}
    </Stack>
  );
};
