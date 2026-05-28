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
                width: '100%',
                minWidth: 0,
                alignSelf: 'stretch',
                [theme.breakpoints.up('lg')]: {
                  maxWidth: 1000,
                },
              }),
          p: fullWidth ? { xs: 0.5, sm: 1 } : { xs: 2 },
          mx: fullWidth ? 0 : 'auto',
          gap: fullWidth ? 0 : 2,
          mt: fullWidth ? 0 : 1,
        })}
      >
        {topBar}
        {!!title && <Typography variant="h2">{title}</Typography>}
        {user?.isAuthorized && hasAccess ? (
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
                display: 'flex',
                flexDirection: 'column',
                overflowY: 'auto',
                WebkitOverflowScrolling: 'touch',
              }}
            >
              {children}
            </Box>
          )
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
        {!fullWidth && (
          <>
            <Divider />
            {footer !== undefined ? footer : <Footer />}
          </>
        )}
      </Stack>
    </>
  );
};
