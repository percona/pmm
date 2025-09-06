import { Stack, Typography } from '@mui/material';
import { Page } from 'components/page';
import { FC } from 'react';

export const NotFoundPage: FC = () => {
  return (
    <Page>
      <Stack>
        <Typography>Not found</Typography>
      </Stack>
    </Page>
  );
};
