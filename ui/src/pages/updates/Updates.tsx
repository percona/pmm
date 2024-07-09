import { CardContent, CardMedia, Stack, Typography } from '@mui/material';
import { Card } from '@mui/material';
import { FC } from 'react';
import Welcome from 'assets/welcome.svg';
import { UpdateCard } from './update-card';
import { Messages } from './Updates.messages';
import { Page } from 'components/page';

export const Updates: FC = () => (
  <Page title={Messages.title}>
    <Card>
      <CardMedia sx={{ height: 140 }} image={Welcome} title="green iguana" />
      <CardContent sx={{ p: 3 }}>
        <Stack gap={1}>
          <Typography variant="h3">{Messages.welcome.title}</Typography>
          <Typography variant="body1" color="text.secondary">
            {Messages.welcome.description}
          </Typography>
        </Stack>
      </CardContent>
    </Card>
    <UpdateCard />
  </Page>
);
