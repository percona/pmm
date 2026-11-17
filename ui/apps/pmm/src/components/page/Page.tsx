import { FC } from 'react';
import { PageProps } from './Page.types';
import {
  Alert,
  Box,
  Card,
  CardActions,
  GlobalStyles,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import { PageContainer } from '@percona/peak-ui';
import { useUser } from 'contexts/user';
import { Messages } from './Page.messages';
import { PMM_HOME_URL } from 'lib/constants';
import { Footer } from 'components/footer';
import { updateDocumentTitle } from 'utils/document.utils';
import { Link as RouterLink } from 'react-router-dom';

export const Page: FC<PageProps> = ({
  title,
  hideTitle,
  topBar,
  footer,
  children,
  maxWidth,
  fullWidth,
  surface = 'paper',
  roles,
}) => {
  const { user } = useUser();
  updateDocumentTitle(title);
  const hasAccess = !roles || roles?.some((role) => user?.orgRole === role);
  const content =
    user?.isAuthorized && hasAccess ? (
      children
    ) : (
      <Card variant="outlined" sx={{ p: 2 }}>
        <Alert severity="error" sx={{ mb: 1 }} data-testid="unauthorized">
          {Messages.noAcccess}
        </Alert>
        <CardActions>
          <Typography>
            {Messages.goBack}
            <Link to={PMM_HOME_URL} component={RouterLink}>
              {Messages.home}
            </Link>
          </Typography>
        </CardActions>
      </Card>
    );

  return (
    <>
      <GlobalStyles
        styles={(theme) => ({
          'html, body': {
            backgroundColor:
              surface === 'paper'
                ? theme.palette.background.paper
                : theme.palette.background.default,
          },
        })}
      />
      {fullWidth ? (
        <Stack
          sx={{
            flex: 1,
            minHeight: 0,
            width: '100%',
            maxWidth: '100%',
            height: '100%',
            maxHeight: '100%',
            minWidth: 0,
            alignSelf: 'stretch',
            alignItems: 'stretch',
            overflow: 'hidden',
            boxSizing: 'border-box',
            p: { xs: 0.5, sm: 1 },
          }}
        >
          {topBar}
          {!!title && !hideTitle && (
            <Typography variant="h2">{title}</Typography>
          )}
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
            {content}
          </Box>
        </Stack>
      ) : (
        <PageContainer maxWidth={maxWidth}>
          {topBar}
          {!!title && !hideTitle && (
            <Typography variant="h2">{title}</Typography>
          )}
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
            {content}
          </Box>
          {footer !== undefined ? footer : <Footer />}
        </PageContainer>
      )}
    </>
  );
};
