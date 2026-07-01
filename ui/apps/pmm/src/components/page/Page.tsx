import { FC } from 'react';
import { PageProps } from './Page.types';
import {
  Alert,
  Box,
  Card,
  CardActions,
  Divider,
  GlobalStyles,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import { useUser } from 'contexts/user';
import { Messages } from './Page.messages';
import { PMM_HOME_URL } from 'lib/constants';
import { Footer } from 'components/footer';
import { updateDocumentTitle } from 'utils/document.utils';
import { Link as RouterLink } from 'react-router-dom';

export const Page: FC<PageProps> = ({
  title,
  topBar,
  footer,
  children,
  fullWidth,
  surface,
  roles,
}) => {
  const { user } = useUser();
  updateDocumentTitle(title);
  const hasAccess = !roles || roles?.some((role) => user?.orgRole === role);

  return (
    <>
      {surface && (
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
      )}
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
          gap: 2,
          mt: 1,
        }}
      >
        {topBar}
        {!!title && <Typography variant="h2">{title}</Typography>}
        <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          {user?.isAuthorized && hasAccess ? (
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
          )}
        </Box>
        {footer !== null && <Divider />}
        {footer !== undefined ? footer : <Footer />}
      </Stack>
    </>
  );
};
