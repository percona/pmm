import { CardContent, CardMedia, Stack, Typography } from '@mui/material';
import { Card } from '@mui/material';
import { FC } from 'react';
import Welcome from 'assets/welcome.svg';
import { UpdateCard } from './update-card';
import { Messages } from './Updates.messages';

export const Updates: FC = () => {
  return (
    <Stack
      sx={{
        width: 800,
        py: 3,
        mx: 'auto',
        gap: 3,
      }}
    >
      <Typography variant="h2">{Messages.title}</Typography>
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
    </Stack>
  );
};
