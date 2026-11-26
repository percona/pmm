import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import { useRealtimeSessions } from 'hooks/api/useRealtime';
import { FC } from 'react';
import { Navigate } from 'react-router-dom';

const RealtimeTab: FC = () => {
  const { data: sessions, isLoading } = useRealtimeSessions();

  if (isLoading) {
    return (
      <Stack
        data-testid="realtime-tab-loading"
        alignItems="center"
        justifyContent="center"
        height="100%"
      >
        <CircularProgress />
      </Stack>
    );
  }

  if (sessions?.length) {
    return <Navigate to="/rta/sessions" />;
  }

  return <Navigate to="/rta/selection" />;
};

export default RealtimeTab;
