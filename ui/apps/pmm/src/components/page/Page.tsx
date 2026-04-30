import { FC } from 'react';
import { PageProps } from './Page.types';
import {
  Alert,
  Box,
  Card,
  CardActions,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import { useUser } from 'contexts/user';
import { Messages } from './Page.messages';
import { PMM_HOME_URL } from 'lib/constants';
import { Footer } from 'components/footer';
import { updateDocumentTitle } from 'utils/document.utils';

export const Page: FC<PageProps> = ({ title, topBar, footer, children, fullWidth }) => {
  const { user } = useUser();
  updateDocumentTitle(title);

  return (
    <Stack
      sx={(theme) => ({
        flex: 1,
        minHeight: 0,
        flexDirection: 'column',
        ...(fullWidth
          ? {
              width: '100%',
              maxWidth: '100%',
              height: '100%',
              maxHeight: '100%',
              minWidth: 0,
              alignSelf: 'stretch',
              alignItems: 'stretch',
              overflow: 'hidden',
              boxSizing: 'border-box',
            }
          : {
              [theme.breakpoints.up('lg')]: {
                width: 1000,
              },
              width: {
                md: 'auto',
              },
            }),
        p: fullWidth
          ? { xs: 0.5, sm: 1 }
          : {
              xs: 2,
            },
        mx: fullWidth ? 0 : 'auto',
        gap: fullWidth ? 0 : 3,
        mt: fullWidth ? 0 : 1,
      })}
    >
      {topBar}
      {!!title && <Typography variant="h2">{title}</Typography>}
      {user?.isAuthorized ? (
        fullWidth ? (
          <Box
            sx={{
              flex: 1,
              minHeight: 0,
              minWidth: 0,
              height: '100%',
              maxHeight: '100%',
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
            }}
          >
            {children}
          </Box>
        ) : (
          <Box
            sx={{
              flex: 1,
              minHeight: 0,
              minWidth: 0,
              overflowY: 'auto',
              WebkitOverflowScrolling: 'touch',
            }}
          >
            {children}
          </Box>
        )
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
      {footer !== undefined ? footer : <Footer />}
    </Stack>
  );
};
