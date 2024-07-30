import {
  Alert,
  CardContent,
  CardMedia,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import { Card } from '@mui/material';
import { FC } from 'react';
import Welcome from 'assets/welcome.svg';
import { UpdateCard } from './update-card';
import { Messages } from './Updates.messages';
import { Page } from 'components/page';
import { useSettings } from 'hooks/api/useSettings';
import { PMM_SETTINGS_URL } from 'constants';
import { UpdateFooter } from './update-footer';

export const Updates: FC = () => {
  const { data: settings } = useSettings();

  return (
    <Page title={Messages.title} footer={<UpdateFooter />}>
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
      {settings?.updatesEnabled ? (
        <UpdateCard />
      ) : (
        <Card>
          <CardContent>
            <Stack gap={1}>
              <Alert severity="warning">{Messages.disabled.title}</Alert>
              <Typography variant="body1" color="text.secondary">
                {Messages.disabled.description}
                <Link href={PMM_SETTINGS_URL}>
                  {Messages.disabled.settings}
                </Link>
                .
              </Typography>
            </Stack>
          </CardContent>
        </Card>
      )}
    </Page>
  );
};
