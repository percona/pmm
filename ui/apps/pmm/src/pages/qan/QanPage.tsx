import { Alert, Box, Link } from '@mui/material';
import { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { Page } from 'components/page';
import { useSettings } from 'contexts/settings';
import { PMM_NEW_NAV_GRAFANA_PATH, PMM_SETTINGS_URL } from 'lib/constants';
import { QanLayout } from './components/QanLayout';
import { QanPanelProvider } from './hooks/QanPanelProvider';

const QanPage: FC = () => {
  const { settings } = useSettings();
  const nativeEnabled = settings?.nativeQanEnabled ?? false;

  return (
    <Page title="" fullWidth footer={null}>
      <QanPanelProvider>
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            flex: 1,
            minHeight: 0,
            height: '100%',
            overflow: 'hidden',
          }}
        >
          {!nativeEnabled ? (
            <Alert severity="warning" sx={{ mx: 3, mt: 1, flexShrink: 0 }}>
              Native Query Analytics is a Technical Preview. Enable it under{' '}
              <Link component={RouterLink} to={PMM_SETTINGS_URL}>
                Settings → Advanced
              </Link>{' '}
              or continue using{' '}
              <Link
                href={`${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics`}
                underline="hover"
              >
                Grafana QAN
              </Link>
              .
            </Alert>
          ) : null}
          <QanLayout />
        </Box>
      </QanPanelProvider>
    </Page>
  );
};

export default QanPage;
