import { Card, Stack } from '@mui/material';
import { Page } from 'components/page';
import { FC } from 'react';

export const AlertsPage: FC = () => {
  return (
    <Page title="Alerts">
      <Card>
        <Stack
          sx={{
            iframe: {
              border: 'none',
              height: '100vh',
            },
          }}
        >
          <iframe src="/graph/alerting"></iframe>
        </Stack>
      </Card>
    </Page>
  );
};
