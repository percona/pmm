import { FC } from 'react';
import { PageProps } from './Page.types';
import {
  Alert,
  Card,
  CardActions,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import { useUser } from 'contexts/user';
import { Messages } from './Page.messages';
import { PMM_HOME_URL } from 'constants';
import { Footer } from 'components/footer';

export const Page: FC<PageProps> = ({ title, footer, children }) => {
  const { user } = useUser();

  return (
    <Stack
      sx={{
        width: {
          sm: 1000,
          xs: 'auto',
        },
        p: {
          xs: 2,
        },
        mx: 'auto',
        gap: 3,
      }}
    >
      {!!title && <Typography variant="h2">{title}</Typography>}
      {user?.isAuthorized ? (
        children
      ) : (
        <Card sx={{ p: 2 }}>
          <Alert severity="error" sx={{ mb: 1 }}>
            {Messages.noAcccess}
          </Alert>
          <CardActions>
            <Typography>
              {Messages.goBack}
              <Link href={PMM_HOME_URL}>{Messages.home}</Link>
            </Typography>
          </CardActions>
        </Card>
      )}
      {footer ? footer : <Footer />}
    </Stack>
  );
};
