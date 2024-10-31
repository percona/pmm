import { Stack } from '@mui/material';
import { Page } from 'components/page';
import { FC } from 'react';
import { useLocation } from 'react-router-dom';

export const DashboardsPage: FC = () => {
  const location = useLocation();

  return (
    <Page title="PostgreSQL Instances Overview">
      <Stack
        sx={(theme) => ({
          iframe: {
            border: '1px solid rgb(213, 215, 217)',
            height: '100vh',
            borderRadius: theme.shape.borderRadius / 2,
          },
        })}
      >
        <iframe src={`/graph${location.pathname}`} seamless></iframe>
      </Stack>
    </Page>
  );
};
