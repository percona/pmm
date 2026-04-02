import { FC } from 'react';
import { PageProps } from './Page.types';
import {
  Alert,
  Box,
  Card,
  CardActions,
  Divider,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import { useUser } from 'contexts/user';
import { Messages } from './Page.messages';
import { PMM_HOME_URL } from 'lib/constants';
import { Footer } from 'components/footer';
import { updateDocumentTitle } from 'utils/document.utils';

export const Page: FC<PageProps> = ({
  title,
  topBar,
  footer,
  children,
  fullWidth,
}) => {
  const { user } = useUser();
  updateDocumentTitle(title);

  return (
    <Stack
      sx={{
        flex: 1,
        width: '100%',
        maxWidth: {
          lg: 1000,
        },
        p: {
          xs: 2,
        },
        px: {
          md: fullWidth ? 4 : undefined,
        },
        mx: 'auto',
        gap: 3,
        mt: 1,
      }}
    >
      {topBar}
      {!!title && <Typography variant="h2">{title}</Typography>}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {user?.isAuthorized ? (
          children
        ) : (
          <Card variant="outlined" sx={{ p: 2 }}>
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
      </Box>
      <Divider />
      {footer !== undefined ? footer : <Footer />}
    </Stack>
  );
};
