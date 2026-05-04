import { CircularProgress, Stack } from '@mui/material';
import { GrafanaPage } from 'pages/grafana';
import { useBootstrap } from 'hooks/utils/useBootstrap';

/**
 * Full-viewport Grafana iframe without PMM shell (sidebar/header).
 * Used for /graph/login so session expiry after password change does not show duplicate nav (PMM-14971).
 */
export const GrafanaFullPageLogin = () => {
  const { isReady } = useBootstrap();

  if (!isReady) {
    return (
      <Stack
        alignItems="center"
        justifyContent="center"
        sx={{
          flex: 1,
          minHeight: '100vh',
          padding: 10,
        }}
      >
        <CircularProgress data-testid="pmm-grafana-login-fullpage-loading" />
      </Stack>
    );
  }

  return (
    <Stack direction="column" flex={1} sx={{ minHeight: '100vh' }}>
      <GrafanaPage />
    </Stack>
  );
};
